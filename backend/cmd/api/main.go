package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	"solv-backend/internal/infrastructure/storage/memory"
	"solv-backend/internal/infrastructure/storage/postgres"
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

	authService := services.NewAuthService(db, tenantRepo)

	astAnalyzer := services.NewStaticASTAnalyzer()
	dockerRunner := docker.NewDockerEvaluationRunner(cli)
	semgrepWorker := services.NewSemgrepWorker(workspaceRepo, dockerClient, "internal/infrastructure/semgrep/rules")
	evaluationService := services.NewEvaluationService(exerciseRepo, astAnalyzer, semgrepWorker, dockerRunner)
	templateRepo := postgres.NewPostgresTemplateRepository(db.GetDB())
	workspaceService := services.NewWorkspaceService(workspaceRepo, dockerClient, hostMonitor).
		WithProvisioning(subjectRepo, templateRepo, db.GetDB())
	subjectService := services.NewSubjectService(subjectRepo)
	submissionService := services.NewSubmissionService(submissionRepo)
	teacherInvService := services.NewTeacherInvitationService(teacherInvRepo)

	zombieCollector := services.NewZombieCollectorWorker(workspaceRepo, dockerClient, 30*time.Second)

	qosWorker := services.NewQoSOrchestratorWorker(workspaceRepo, dockerClient, hostMonitor, 15*time.Minute, 10*time.Second)

	serverPoliciesService := services.NewServerPoliciesService(tenantRepo)
	workspaceService.SetPoliciesService(serverPoliciesService)
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

	wsHub := httpdelivery.NewWebSocketHub()
	go wsHub.Run()

	wsHandler := httpdelivery.NewWebSocketHandler(wsHub, authService)

	adminHandler := httpdelivery.NewAdminHandler(auditLogRepo, tenantRepo, workspaceRepo, subjectRepo, hostMonitor)
	serverPoliciesHandler := httpdelivery.NewServerPoliciesHandler(serverPoliciesService)
	tenantLogoHandler := httpdelivery.NewTenantLogoHandler(tenantRepo)
	adminHandler.SetOrchestrator(dockerClient)
	studentHandler := httpdelivery.NewStudentHandler(subjectRepo, workspaceRepo, submissionRepo, exerciseRepo)

	teacherRepo := postgres.NewPostgresTeacherRepository(db.GetDB())
	teacherService := services.NewTeacherService(teacherRepo, submissionRepo)
	teacherService.SetEvaluationService(evaluationService)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService)

	// Slice 14: Periodos Académicos, Modo Mantenimiento, Reasignación y Estudiantes
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())
	govService := services.NewAdminGovernanceService(subjectRepo, govRepo)
	govService.SetAuditRepo(auditLogRepo)
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

	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)
	imageVerificationService := services.NewImageVerificationService(cli)
	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, govService).
		WithImageService(imageVerificationService).
		WithAuditLogRepo(auditLogRepo)
	maintenanceMiddleware := httpdelivery.MaintenanceMiddleware(tenantRepo)

	templateAuditWorker := services.NewTemplateAuditWorker(cli, govRepo)
	templateAuditWorker.Start(ctx)

	// Worker cron cada 24h para archivado automático de periodos expirados
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			count, err := academicPeriodService.ArchiveExpiredPeriods(context.Background())
			if err != nil {
				log.Printf("Error in automatic archiving of academic periods: %v", err)
			} else if count > 0 {
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

	// Slice 16: Backups Configurables y Retención (ADR-035)
	backupRepo := postgres.NewPostgresBackupRepository(db.GetDB())
	backupService := services.NewBackupService(backupRepo, notificationService, "")
	backupHandler := httpdelivery.NewBackupHandler(backupService)

	// Prueba asíncrona de entorno para plantillas (DA-01, DA-06)
	envTestJobRepo := memory.NewEnvTestJobMemoryRepository(2 * time.Hour)
	envTestAdapter := docker.NewEnvTestDockerAdapter(cli, 60*time.Second)
	envTestService := services.NewEnvTestService(envTestJobRepo, envTestAdapter, envTestAdapter, services.EnvTestConfig{
		MaxConcurrentJobs: 2,
		TotalJobTimeout:   15 * time.Minute,
		MemoryLimitMB:     256,
	})
	envTestHandler := httpdelivery.NewEnvTestHandler(envTestService)

	// Juez de ejercicios: perfiles de lenguaje versionados y auditoría
	langProfileRepo := postgres.NewPostgresLanguageProfileRepository(db.GetDB())
	langProfileService := services.NewLanguageProfileService(langProfileRepo)
	langProfileHandler := httpdelivery.NewLanguageProfileHandler(langProfileService)

	handlersStruct := httpdelivery.Handlers{
		UserHandler:              httpdelivery.NewUserHandler(db, v),
		TemplateHandler:          httpdelivery.NewTemplateHandler(db, v),
		AuthHandler:              httpdelivery.NewAuthHandler(authService),
		EvaluationHandler:        evalHandler,
		LanguageProfileHandler:   langProfileHandler,
		WorkspaceHandler:         httpdelivery.NewWorkspaceHandler(workspaceService, v),
		MetricsHandler:           httpdelivery.NewMetricsHandler(workspaceRepo, hostMonitor, zombieCollector),
		ConfigHandler:            httpdelivery.NewConfigHandler(tenantRepo),
		SubjectHandler:           httpdelivery.NewSubjectHandler(subjectService),
		SubmissionHandler:        subHandler,
		TeacherInvitationHandler: httpdelivery.NewTeacherInvitationHandler(teacherInvService),
		ClassroomHandler:         httpdelivery.NewClassroomHandler(),
		AdminHandler:             adminHandler,
		ServerPoliciesHandler:    serverPoliciesHandler,
		TenantLogoHandler:        tenantLogoHandler,
		AdminAcademicHandler:     adminAcademicHandler,
		StudentHandler:           studentHandler,
		TeacherHandler:           teacherHandler,
		NotificationHandler:      notificationHandler,
		BackupHandler:            backupHandler,
		WebSocketHandler:         wsHandler,
		EnvTestHandler:           envTestHandler,
		TenantMiddleware:         tenantMiddleware,
		MaintenanceMiddleware:    maintenanceMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlersStruct)

	// Aplicar MaintenanceMiddleware y CORS
	handler := httpdelivery.WithCORS(maintenanceMiddleware(mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
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
