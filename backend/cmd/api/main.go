package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/delivery/http/middleware"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/docker"
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

	if err := db.RunInitialMigrations(); err != nil {
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
	workspaceService := services.NewWorkspaceService(workspaceRepo, dockerClient, hostMonitor)
	subjectService := services.NewSubjectService(subjectRepo)
	submissionService := services.NewSubmissionService(submissionRepo)
	teacherInvService := services.NewTeacherInvitationService(teacherInvRepo)

	zombieCollector := services.NewZombieCollectorWorker(workspaceRepo, dockerClient, 30*time.Second)

	qosWorker := services.NewQoSOrchestratorWorker(workspaceRepo, dockerClient, hostMonitor, 15*time.Minute, 10*time.Second)
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

	adminHandler := httpdelivery.NewAdminHandler(auditLogRepo, tenantRepo, workspaceRepo)
	studentHandler := httpdelivery.NewStudentHandler(subjectRepo, workspaceRepo, submissionRepo, exerciseRepo)

	teacherRepo := postgres.NewPostgresTeacherRepository(db.GetDB())
	teacherService := services.NewTeacherService(teacherRepo, submissionRepo)
	teacherService.SetEvaluationService(evaluationService)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService)

	// Slice 14: Periodos Académicos, Modo Mantenimiento, Reasignación y Estudiantes
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())
	govService := services.NewAdminGovernanceService(subjectRepo, govRepo)

	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)
	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, govService)
	maintenanceMiddleware := httpdelivery.MaintenanceMiddleware(tenantRepo)

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

	handlersStruct := httpdelivery.Handlers{
		UserHandler:              httpdelivery.NewUserHandler(db, v),
		TemplateHandler:          httpdelivery.NewTemplateHandler(db, v),
		AuthHandler:              httpdelivery.NewAuthHandler(authService),
		EvaluationHandler:        evalHandler,
		WorkspaceHandler:         httpdelivery.NewWorkspaceHandler(workspaceService, v),
		MetricsHandler:           httpdelivery.NewMetricsHandler(workspaceRepo, hostMonitor, zombieCollector),
		ConfigHandler:            httpdelivery.NewConfigHandler(tenantRepo),
		SubjectHandler:           httpdelivery.NewSubjectHandler(subjectService),
		SubmissionHandler:        subHandler,
		TeacherInvitationHandler: httpdelivery.NewTeacherInvitationHandler(teacherInvService),
		ClassroomHandler:         httpdelivery.NewClassroomHandler(),
		AdminHandler:             adminHandler,
		AdminAcademicHandler:     adminAcademicHandler,
		StudentHandler:           studentHandler,
		TeacherHandler:           teacherHandler,
		NotificationHandler:      notificationHandler,
		WebSocketHandler:         wsHandler,
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
