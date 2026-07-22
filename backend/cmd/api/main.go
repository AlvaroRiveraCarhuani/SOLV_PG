package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/client"
	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
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

	labInstanceRepo := postgres.NewPostgresLabInstanceRepository(db.GetDB())
	templateRepo := postgres.NewPostgresTemplateRepository(db.GetDB())
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	workspaceRepo := postgres.NewPostgresWorkspaceRepository(db.GetDB())

	labService := services.NewLabService(labInstanceRepo, templateRepo, dockerClient)
	authService := services.NewAuthService(db)

	astAnalyzer := services.NewStaticASTAnalyzer()
	dockerRunner := docker.NewDockerEvaluationRunner(cli)
	evaluationService := services.NewEvaluationService(exerciseRepo, astAnalyzer, dockerRunner)
	workspaceService := services.NewWorkspaceService(workspaceRepo, dockerClient, hostMonitor)

	qosWorker := services.NewQoSOrchestratorWorker(workspaceRepo, dockerClient, hostMonitor, 15*time.Minute, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	qosWorker.Start(ctx)

	v := validator.New()

	handlersStruct := httpdelivery.Handlers{
		UserHandler:       httpdelivery.NewUserHandler(db, v),
		TemplateHandler:   httpdelivery.NewTemplateHandler(db, v),
		AuthHandler:       httpdelivery.NewAuthHandler(authService),
		LabHandler:        httpdelivery.NewLabHandler(labService, v),
		EvaluationHandler: httpdelivery.NewEvaluationHandler(evaluationService, v),
		WorkspaceHandler:  httpdelivery.NewWorkspaceHandler(workspaceService, v),
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlersStruct)

	handler := httpdelivery.WithCORS(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
