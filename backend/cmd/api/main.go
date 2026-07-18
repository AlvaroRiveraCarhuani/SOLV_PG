package main

import (
	"log"
	"net/http"
	"os"

	"solv-backend/internal/application"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/docker"

	"github.com/docker/docker/client"
	"github.com/go-playground/validator/v10"
	_ "github.com/lib/pq"
)

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Fatal: failed to initialize docker client: %v", err)
	}
	defer cli.Close()

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Fatal: failed to initialize docker service client: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("Fatal: DATABASE_URL environment variable is required")
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		log.Fatalf("Fatal: failed to initialize postgres database: %v", err)
	}

	if err := db.RunInitialMigrations(); err != nil {
		log.Fatalf("Fatal: failed to run database migrations: %v", err)
	}

	labService := application.NewLabService(db, dockerClient)

	v := validator.New()

	handlers := httpdelivery.Handlers{
		UserHandler:     httpdelivery.NewUserHandler(db, v),
		TemplateHandler: httpdelivery.NewTemplateHandler(db, v),
		LabHandler:      httpdelivery.NewLabHandler(labService, v),
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)
	handler := httpdelivery.WithCORS(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
