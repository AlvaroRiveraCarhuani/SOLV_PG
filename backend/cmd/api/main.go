package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/delivery/http/middleware"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/docker"
	"solv-backend/internal/infrastructure/mailer"
	"solv-backend/internal/infrastructure/storage/memory"
	"solv-backend/internal/infrastructure/storage/postgres"
	"solv-backend/internal/infrastructure/storage/seaweedfs"
	"solv-backend/internal/infrastructure/system"
)

func main() {
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := database.NewPostgresDB(dbDSN)
	if err != nil {
		log.Fatalf("Fatal: failed to connect to Postgres: %v", err)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Fatal: failed to initialize Docker client: %v", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Fatal: failed to initialize raw docker SDK client: %v", err)
	}

	// Determinar directorio de migraciones relativo al ejecutable.
	// En dev (go run) __file__ apunta al source; en prod el binario
	// se compila en /app y las migraciones se copian a /app/migrations.
	migrationsDir := resolveMigrationsDir()
	if err := database.RunMigrations(db.GetDB().DB, migrationsDir); err != nil {
		log.Fatalf("Fatal: failed to run database migrations: %v", err)
	}

	hostMonitor := system.NewGopsutilHostMonitor(15.0)

	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	workspaceRepo := postgres.NewPostgresWorkspaceRepository(db.GetDB())
	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	submissionRepo := postgres.NewPostgresSubmissionRepository(db.GetDB())
	teacherInvRepo := postgres.NewPostgresTeacherInvitationRepository(db.GetDB())
	teacherRepo := postgres.NewPostgresTeacherRepository(db.GetDB())

	authService := services.NewAuthService(db, tenantRepo).WithTeacherInvitationRepo(teacherInvRepo)

	astAnalyzer := services.NewStaticASTAnalyzer()
	dockerRunner := docker.NewDockerEvaluationRunner(cli)
	semgrepWorker := services.NewSemgrepWorker(workspaceRepo, dockerClient, "internal/infrastructure/semgrep/rules")
	evaluationService := services.NewEvaluationService(exerciseRepo, astAnalyzer, semgrepWorker, dockerRunner)
	generatedCaseSnapshotRepo := postgres.NewPostgresGeneratedCaseSnapshotRepository(db.GetDB())
	evaluationService.SetGeneratedCaseSnapshotRepository(generatedCaseSnapshotRepo)
	scriptSandboxRunner := docker.NewDockerScriptSandboxRunner(cli)
	evaluationService.SetScriptSandboxRunner(scriptSandboxRunner)
	templateRepo := postgres.NewPostgresTemplateRepository(db.GetDB())
	workspaceService := services.NewWorkspaceService(workspaceRepo, dockerClient, hostMonitor).
		WithProvisioning(subjectRepo, templateRepo).
		WithExerciseSubmission(exerciseRepo, submissionRepo)
	subjectService := services.NewSubjectService(subjectRepo)
	keystrokeRepo := postgres.NewPostgresKeystrokeRepository(db.GetDB())
	submissionService := services.NewSubmissionService(submissionRepo)
	submissionService.SetKeystrokeRepository(keystrokeRepo)
	submissionService.SetGeneratedCaseSnapshotRepository(generatedCaseSnapshotRepo)
	smtpMailer := mailer.NewSMTPMailerFromEnv()
	defer func() { _ = smtpMailer.Close() }()
	teacherInvService := services.NewTeacherInvitationService(teacherInvRepo, smtpMailer)

	zombieCollector := services.NewZombieCollectorWorker(workspaceRepo, dockerClient, 30*time.Second)

	qosWorker := services.NewQoSOrchestratorWorker(workspaceRepo, dockerClient, hostMonitor, 15*time.Minute, 10*time.Second)

	// Preview efímero de SpeedGrader: el QoS worker es su único dueño temporal.
	reviewEnvRepo := postgres.NewPostgresReviewEnvironmentRepository(db.GetDB())
	reviewEnvService := services.NewReviewEnvironmentService(reviewEnvRepo, teacherRepo, dockerClient, hostMonitor)
	qosWorker.SetReviewEnvironmentReaper(reviewEnvService.ReapExpired)

	serverPoliciesService := services.NewServerPoliciesService(tenantRepo)
	workspaceService.SetPoliciesService(serverPoliciesService)
	// Scale-to-Zero de motores de base de datos auto-orquestados: el worker de QoS apaga
	// los satélites (mysql, mariadb, mongodb, redis) que ya no usa ningún laboratorio.
	qosWorker.SetManagedSatelliteReconciler(workspaceService.ReconcileIdleManagedSatellitesScheduled)
	qosWorker.SetPoliciesProvider(func(ctx context.Context) (int, bool) {
		policies, err := serverPoliciesService.Get(ctx, domain.DefaultTenantID)
		if err != nil {
			return 0, false
		}
		return policies.InactivityMinutes, true
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	qosWorker.Start(ctx)
	go zombieCollector.Start(ctx)

	v := validator.New()

	jwtSecret := strings.Trim(strings.TrimSpace(os.Getenv("JWT_SECRET")), `"`)
	tenantMiddleware := middleware.WithTenant(tenantRepo, []byte(jwtSecret))

	auditLogRepo := postgres.NewAuditLogRepository(db.GetDB())

	// Reconciliacion bidireccional (D2, ADR-049): el collector saca contenedores
	// sin registro; el reconciler marca registros sin contenedor (orphaned) y
	// daemon caido (unknown), dentro del mismo ciclo de 30 s del collector.
	stateReconciler := services.NewWorkspaceStateReconciler(workspaceRepo, dockerClient)
	stateReconciler.SetAuditRepository(auditLogRepo)
	zombieCollector.SetStateReconciler(stateReconciler)

	wsHub := httpdelivery.NewWebSocketHub()
	go wsHub.Run()

	wsHandler := httpdelivery.NewWebSocketHandler(wsHub, authService)

	adminHandler := httpdelivery.NewAdminHandler(auditLogRepo, tenantRepo, workspaceRepo, subjectRepo, hostMonitor)
	// Telemetria unica del host para el panel de salud: mide /proc via
	// gopsutil y el daemon de Docker, desacoplada de la capa HTTP. El mismo
	// origen alimenta al sampler persistido (D1) y al handler del panel.
	hostTelemetry := system.NewSystemTelemetry("/var/lib/docker", dockerClient)
	adminHandler.SetTelemetry(hostTelemetry)

	// Historial persistido del host (D1, ADR-049): el grafico ya no se siembra en
	// memoria; un worker escribe 1 fila/minuto y poda a 7 dias, y el panel lee
	// los ultimos samples reales desde la base tras un reinicio.
	metricsSampleRepo := postgres.NewPostgresHostMetricsSampleRepository(db.GetDB())
	hostMetricsSampler := services.NewHostMetricsSampler(metricsSampleRepo, hostTelemetry, func(ctx context.Context) (int, error) {
		running, err := workspaceRepo.GetAllRunningWorkspaces(ctx)
		if err != nil {
			return 0, err
		}
		return len(running), nil
	})
	go hostMetricsSampler.Start(ctx)
	adminHandler.SetMetricsSampleStore(metricsSampleRepo)
	adminHandler.SetPoliciesService(serverPoliciesService)
	serverPoliciesHandler := httpdelivery.NewServerPoliciesHandler(serverPoliciesService)
	// Módulo de IA Híbrida (Nube + Servidor Local LAN) gobernado por institución:
	// cifrado AES-256-GCM de la API Key, escudo anti-SSRF y cuota horaria por usuario.
	aiSettingsService := services.NewAISettingsService(tenantRepo)
	aiSettingsHandler := httpdelivery.NewAISettingsHandler(aiSettingsService)
	// Slice 14: Periodos Académicos, Modo Mantenimiento, Reasignación y Estudiantes
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())
	// Directorio de estudiantes para mostrar identidades reales (nombre/correo)
	// en los paneles admin, nunca UUIDs internos.
	adminHandler.SetStudentDirectory(govRepo)
	evaluationService.SetAdminGovernanceRepository(govRepo)

	// Copiloto estructurado: sugerencia de componentes (admin) y borradores de
	// ejercicio con sello determinista del juez (docente) sobre CompleteJSON, con
	// grounding del catálogo real vía govRepo.
	aiCopilotService := services.NewAICopilotService(aiSettingsService, dockerRunner, govRepo)
	aiCopilotHandler := httpdelivery.NewAICopilotHandler(aiCopilotService)
	tenantLogoHandler := httpdelivery.NewTenantLogoHandler(tenantRepo)
	adminHandler.SetOrchestrator(dockerClient)
	studentHandler := httpdelivery.NewStudentHandler(subjectRepo, workspaceRepo, submissionRepo, exerciseRepo, teacherRepo)

	// Aula de la clase: sesión, presencia y tablero. El ciclo de vida del entorno
	// se enlaza al aula cuando nace dentro de una sesión abierta.
	classSessionRepo := postgres.NewPostgresClassSessionRepository(db.GetDB())
	classSessionService := services.NewClassSessionService(classSessionRepo)
	classSessionHandler := httpdelivery.NewClassSessionHandler(classSessionService)
	workspaceService.WithClassSessions(classSessionRepo)

	// Congelamiento del examen (ADR-005 §5): al cerrar el aula de un examen, el
	// volumen del alumno se reconstruye montado en solo lectura. El worker cubre
	// el cierre por vencimiento y lo que quedó pendiente con el backend apagado.
	examFreezeRepo := postgres.NewPostgresExamFreezeRepository(db.GetDB())
	examFreezeService := services.NewExamFreezeService(examFreezeRepo, classSessionRepo, workspaceRepo, workspaceService)
	examFreezeService.SetAuditRepository(auditLogRepo)
	classSessionService.WithExamFreezer(examFreezeService)
	// Entrega anticipada: el alumno que termina antes cierra solo su volumen y
	// la clase sigue escribiendo para los demás (ADR-005 §5.2).
	workspaceService.SetExamSubmissionFreezer(examFreezeService)
	go services.NewExamFreezeWorker(examFreezeService).Start(ctx)

	// Avance voluntario del docente (ADR-058). Emitir y congelar viven en el
	// servicio; la aplicación en el entorno del alumno necesita el snapshots de
	// SeaweedFS y la capacidad de escribir en el volumen sin pausar el contenedor.
	classAdvanceRepo := postgres.NewPostgresClassAdvanceRepository(db.GetDB(), classSessionRepo)
	classAdvanceService := services.NewClassAdvanceService(classAdvanceRepo, classSessionRepo)
	classAdvanceService.SetAuditRepository(auditLogRepo)
	classAdvanceService.SetHistoryStore(classAdvanceRepo)
	var advanceVolumes domain.WorkspaceVolumeStore = dockerClient
	if snapshots := newAdvanceSnapshotStore(ctx); snapshots != nil {
		classAdvanceService.WithApplier(snapshots, workspaceRepo, advanceVolumes)
		classAdvanceService.SetApplyConcurrency(envInt("ADVANCE_APPLY_CONCURRENCY", 4))
		classAdvanceService.SetRetentionStore(classAdvanceRepo)
		retentionWorker := services.NewAdvanceRetentionWorker(classAdvanceService)
		go retentionWorker.Start(ctx)
	} else {
		log.Printf("[Advance] SeaweedFS no está configurado: los avances no se podrán aplicar ni retener")
	}
	classAdvanceHandler := httpdelivery.NewClassAdvanceHandler(classAdvanceService)

	teacherService := services.NewTeacherService(teacherRepo, submissionRepo)
	teacherService.SetClassSessionStore(classSessionRepo)
	teacherService.SetKeystrokeRepository(keystrokeRepo)
	teacherService.SetEvaluationService(evaluationService)
	teacherService.SetMailer(smtpMailer)
	teacherService.SetTenantRepository(tenantRepo)
	teacherService.SetAuditRepository(auditLogRepo)
	metricsSnapshotService := services.NewMetricsSnapshotService(workspaceRepo, dockerClient, 5*time.Second)
	metricsSnapshotService.Start(ctx)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService, metricsSnapshotService)
	teacherHandler.SetReviewEnvironmentService(reviewEnvService)
	govService := services.NewAdminGovernanceService(subjectRepo, govRepo)
	govService.SetAuditRepo(auditLogRepo)

	// Pedido de entorno del docente: intención + receta armada, sin imágenes ni
	// cuotas técnicas en la vista del docente.
	environmentRequestRepo := postgres.NewPostgresEnvironmentRequestRepository(db.GetDB())
	environmentRequestService := services.NewEnvironmentRequestService(environmentRequestRepo, govRepo)
	environmentRequestHandler := httpdelivery.NewEnvironmentRequestHandler(environmentRequestService)
	// Submódulo 14.7 (ADR-032): executors de docker-prune y reset-pools.
	govService.SetDockerPruner(dockerClient.PruneOrphans)
	govService.SetPoolResetter(func(ctx context.Context) (int64, error) {
		sqlDB := db.GetDB().DB
		stats := sqlDB.Stats()
		idle := int64(stats.Idle)
		restoreIdle := stats.Idle
		if restoreIdle == 0 {
			restoreIdle = 2
		}
		sqlDB.SetMaxIdleConns(0)
		if err := sqlDB.PingContext(ctx); err != nil {
			sqlDB.SetMaxIdleConns(restoreIdle)
			return 0, err
		}
		sqlDB.SetMaxIdleConns(restoreIdle)
		return idle, nil
	})

	// Prueba asíncrona de entorno para plantillas (DA-01, DA-06), pruebas de
	// prácticas IDE y constructor de imágenes. El TTL conserva la última prueba
	// de cada práctica durante la jornada docente.
	envTestJobRepo := memory.NewEnvTestJobMemoryRepository(24 * time.Hour)
	envTestAdapter := docker.NewEnvTestDockerAdapter(cli, 60*time.Second)

	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)
	imageVerificationService := services.NewImageVerificationService(cli)
	templateCacheService := services.NewTemplateCacheService(govRepo, dockerClient)
	templateCacheService.SetImageBuilder(envTestAdapter)
	govService.SetCacheService(templateCacheService)
	govService.SetEnvTestJobRepository(envTestJobRepo)
	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, govService).
		WithImageService(imageVerificationService).
		WithAuditLogRepo(auditLogRepo).
		WithCacheService(templateCacheService)
	maintenanceMiddleware := httpdelivery.MaintenanceMiddleware(tenantRepo)

	templateAuditWorker := services.NewTemplateAuditWorker(cli, govRepo)
	templateAuditWorker.Start(ctx)

	deadlineReminderRepo := postgres.NewPostgresDeadlineReminderRepository(db.GetDB())
	deadlineReminderWorker := services.NewDeadlineReminderWorker(deadlineReminderRepo, tenantRepo, smtpMailer)
	deadlineReminderWorker.Start(ctx, 15*time.Minute)
	defer deadlineReminderWorker.Stop()

	// Auto-recuperación y verificación de disponibilidad de entornos en segundo plano al arrancar el servidor
	go func() {
		warmCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		summary, err := templateCacheService.WarmAllActiveTemplates(warmCtx, "")
		if err != nil {
			log.Printf("[TemplateCache] Advertencia en auto-recuperación de entornos: %v", err)
			return
		}
		// Contadores internos generados por el servicio, no provienen de entrada de usuario.
		//nolint:gosec
		log.Printf("[TemplateCache] Disponibilidad de entornos verificada: %d listos, %d descargados, %d con error (total: %d)",
			summary.AlreadyWarm, summary.WarmedCount, summary.FailedCount, summary.TotalChecked)
	}()

	// Worker cron cada 24h para archivado automático de periodos expirados
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			count, err := academicPeriodService.ArchiveExpiredPeriods(context.Background())
			if err != nil {
				log.Printf("Error in automatic archiving of academic periods: %v", err)
			} else if count > 0 {
				// Contador de base de datos generado por el servicio, no proviene de entrada de usuario.
				//nolint:gosec
				log.Printf("Automatic archiving completed: %d expired periods archived", count)
			}
		}
	}()

	evalHandler := httpdelivery.NewEvaluationHandler(evaluationService, v)
	evalHandler.SetWebSocketHub(wsHub)

	subHandler := httpdelivery.NewSubmissionHandler(submissionService)
	subHandler.SetWebSocketHub(wsHub)

	// Slice 15: Notificaciones Proactivas (ADR-034)
	notificationRepo := postgres.NewPostgresNotificationRepository(db.GetDB())
	notificationService := services.NewNotificationService(notificationRepo, 256)
	defer notificationService.Stop()
	notificationHandler := httpdelivery.NewNotificationHandler(notificationService)
	adminAcademicHandler.WithNotificationService(notificationService)

	// Slice 16: Backups Configurables y Retención (ADR-035)
	backupRepo := postgres.NewPostgresBackupRepository(db.GetDB())
	backupService := services.NewBackupServiceWithContentSource(backupRepo, notificationService, "", postgres.NewPostgresDumpContentSource(db.GetDB()))
	backupService.StartScheduler(ctx, domain.DefaultTenantID)
	backupHandler := httpdelivery.NewBackupHandler(backupService)

	envTestService := services.NewEnvTestService(envTestJobRepo, envTestAdapter, envTestAdapter, services.EnvTestConfig{
		MaxConcurrentJobs: 2,
		TotalJobTimeout:   15 * time.Minute,
		MemoryLimitMB:     256,
	})
	envTestService.SetImageBuilder(envTestAdapter)
	practiceEnvTestService := services.NewPracticeEnvTestService(
		envTestJobRepo,
		envTestAdapter,
		envTestAdapter,
		workspaceService,
		exerciseRepo,
		subjectRepo,
		templateRepo,
		services.PracticeEnvTestConfig{
			MaxConcurrentJobs: 1,
			TotalJobTimeout:   10 * time.Minute,
			SetupTimeout:      5 * time.Minute,
		},
	)
	envTestHandler := httpdelivery.NewEnvTestHandler(envTestService).SetPracticeService(practiceEnvTestService)

	// Juez de ejercicios: perfiles de lenguaje versionados y auditoría
	langProfileRepo := postgres.NewPostgresLanguageProfileRepository(db.GetDB())
	langProfileService := services.NewLanguageProfileService(langProfileRepo)
	langProfileHandler := httpdelivery.NewLanguageProfileHandler(langProfileService)

	// Estructura Curricular: Módulos y Prerrequisitos
	courseModuleRepo := postgres.NewPostgresCourseModuleRepository(db.GetDB())
	courseModuleService := services.NewCourseModuleService(courseModuleRepo, exerciseRepo)
	courseModuleHandler := httpdelivery.NewCourseModuleHandler(courseModuleService)
	evaluationService.SetModuleRepository(courseModuleRepo)

	// Rúbricas de evaluación cualitativa (IDE Persistente)
	rubricRepo := postgres.NewPostgresRubricRepository(db.GetDB())
	rubricService := services.NewDefaultRubricService(rubricRepo, exerciseRepo, db.GetDB())
	rubricService.SetMailer(smtpMailer)
	rubricService.SetTenantRepository(tenantRepo)
	rubricHandler := httpdelivery.NewRubricHandler(rubricService)

	rateLimiter := middleware.NewRateLimiter()
	loginLimiter := middleware.NewUserRateLimiter(5, 5)
	submissionLimiter := middleware.NewUserRateLimiter(10, 10)
	keystrokeLimiter := middleware.NewUserRateLimiter(30, 30)

	handlersStruct := httpdelivery.Handlers{
		UserHandler:                   httpdelivery.NewUserHandler(db, v),
		TemplateHandler:               httpdelivery.NewTemplateHandler(db, v),
		AuthHandler:                   httpdelivery.NewAuthHandler(authService).WithRateLimiter(rateLimiter),
		EvaluationHandler:             evalHandler,
		LanguageProfileHandler:        langProfileHandler,
		CourseModuleHandler:           courseModuleHandler,
		RubricHandler:                 rubricHandler,
		WorkspaceHandler:              httpdelivery.NewWorkspaceHandler(workspaceService, v),
		MetricsHandler:                httpdelivery.NewMetricsHandler(workspaceRepo, hostMonitor, zombieCollector),
		ConfigHandler:                 httpdelivery.NewConfigHandler(tenantRepo),
		SubjectHandler:                httpdelivery.NewSubjectHandler(subjectService),
		SubmissionHandler:             subHandler,
		TeacherInvitationHandler:      httpdelivery.NewTeacherInvitationHandler(teacherInvService),
		ClassroomHandler:              httpdelivery.NewClassroomHandler(),
		ClassSessionHandler:           classSessionHandler,
		ClassAdvanceHandler:           classAdvanceHandler,
		AdminHandler:                  adminHandler,
		ServerPoliciesHandler:         serverPoliciesHandler,
		AISettingsHandler:             aiSettingsHandler,
		AICopilotHandler:              aiCopilotHandler,
		TenantLogoHandler:             tenantLogoHandler,
		AdminAcademicHandler:          adminAcademicHandler,
		StudentHandler:                studentHandler,
		TeacherHandler:                teacherHandler,
		NotificationHandler:           notificationHandler,
		BackupHandler:                 backupHandler,
		WebSocketHandler:              wsHandler,
		EnvTestHandler:                envTestHandler,
		EnvironmentRequestHandler:     environmentRequestHandler,
		TenantMiddleware:              tenantMiddleware,
		MaintenanceMiddleware:         maintenanceMiddleware,
		LoginRateLimitMiddleware:      middleware.EndpointRateLimitMiddleware(loginLimiter, "login"),
		SubmissionRateLimitMiddleware: middleware.EndpointRateLimitMiddleware(submissionLimiter, "submissions"),
		KeystrokeRateLimitMiddleware:  middleware.EndpointRateLimitMiddleware(keystrokeLimiter, "keystroke_events"),
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlersStruct)

	// Aplicar ObservabilityMiddleware, SecurityHeadersMiddleware, CORSMiddleware, GlobalRateLimitMiddleware y MaintenanceMiddleware
	handler := middleware.ObservabilityMiddleware(
		middleware.SecurityHeadersMiddleware(
			middleware.CORSMiddleware(
				middleware.GlobalRateLimitMiddleware(rateLimiter)(
					maintenanceMiddleware(mux),
				),
			),
		),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Puerto obtenido de variable de entorno local, no proviene de peticion HTTP.
	//nolint:gosec
	log.Printf("Server starting on port %s...", port)
	// Sin ReadTimeout/WriteTimeout: hay WebSockets (/api/v1/teacher/live-sessions/{id}/terminal,
	// /ws/v1/evaluations) y descargas largas de backups/logos que exceden cualquier cota razonable.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// resolveMigrationsDir devuelve la ruta al directorio de migraciones.
// Estrategia: primero busca ./migrations (relativo al CWD, funciona con
// "go run" y en el contenedor Docker donde el CWD es /app).
// Si no existe, busca relativo al archivo fuente actual (útil en algunos
// entornos de CI donde el CWD difiere del módulo).
func resolveMigrationsDir() string {
	// Opción de override vía variable de entorno (útil en CI / tests).
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		return v
	}
	// Ruta relativa al CWD: funciona en Docker (/app) y en `go run ./cmd/api`.
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	// Fallback: relativo a este archivo fuente (útil en desarrollo local cuando
	// el CWD no es la raíz del módulo).
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// backend/cmd/api/main.go → backend/migrations
		candidate := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Último recurso: devolver el path relativo y dejar que goose falle con mensaje claro.
	return "migrations"
}

// envInt lee un entero positivo; sin valor válido devuelve el respaldo.
func envInt(name string, fallback int) int {
	if parsed, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name))); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}

// envList lee una lista separada por comas; sin valor devuelve el respaldo por
// defecto que la propia configuración documenta.
func envList(name, fallback string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		raw = fallback
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}

// newAdvanceSnapshotStore arma el almacén de snapshots de avances con la
// configuración de SeaweedFS (ADR-058 §6). Sin configuración devuelve nil: el
// aula sigue funcionando y aplicar un avance responde 503.
func newAdvanceSnapshotStore(ctx context.Context) *seaweedfs.AdvanceSnapshotStore {
	endpoint := strings.TrimSpace(os.Getenv("SEAWEEDFS_ENDPOINT"))
	bucket := strings.TrimSpace(os.Getenv("SEAWEEDFS_BUCKET"))
	accessKey := strings.TrimSpace(os.Getenv("SEAWEEDFS_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("SEAWEEDFS_SECRET_KEY"))
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return nil
	}

	maxBytes := int64(4 << 20)
	if raw := strings.TrimSpace(os.Getenv("ADVANCE_SNAPSHOT_MAX_BYTES")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			maxBytes = parsed
		}
	}

	maxFiles := 512
	if raw := strings.TrimSpace(os.Getenv("ADVANCE_SNAPSHOT_MAX_FILES")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			maxFiles = parsed
		}
	}

	store, err := seaweedfs.NewAdvanceSnapshotStore(ctx, seaweedfs.Config{
		Endpoint:  endpoint,
		Region:    strings.TrimSpace(os.Getenv("SEAWEEDFS_REGION")),
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		MaxBytes:  maxBytes,
		// Las exclusiones del avance viven en configuración (ADR-058 §5).
		Archive: seaweedfs.ArchiveOptions{
			ExcludePrefixes: envList("ADVANCE_SNAPSHOT_EXCLUDES", ".vscode/,.solv_*,.git/,node_modules/,.cache/,.openvscode-server/"),
			MaxFiles:        maxFiles,
		},
	})
	if err != nil {
		log.Printf("[Advance] no se pudo inicializar el almacén de snapshots: %v", err)
		return nil
	}
	return store
}
