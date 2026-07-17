package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"solv-backend/internal/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/docker"

	"github.com/docker/docker/client"
	_ "github.com/lib/pq"
)

type GlobalResponse struct {
	Data    any    `json:"data"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Fatal: failed to initialize docker client: %v", err)
	}
	defer cli.Close()
	dockerManager := docker.NewManager(cli)
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
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var dto domain.CreateUserDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(GlobalResponse{
				Error:   "Invalid JSON body",
				Message: "Cuerpo de la petición inválido",
			})
			return
		}
		if dto.FirstName == "" || dto.LastName == "" || dto.Email == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(GlobalResponse{
				Error:   "Missing required fields",
				Message: "Los campos first_name, last_name y email son obligatorios",
			})
			return
		}
		id, err := db.CreateUser(r.Context(), dto)
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "email already exists" {
				status = http.StatusConflict
			}
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(GlobalResponse{
				Error:   err.Error(),
				Message: "No se pudo crear el usuario",
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(GlobalResponse{
			Data:    map[string]string{"id": id},
			Error:   "",
			Message: "Usuario creado exitosamente",
		})
	})
	mux.HandleFunc("POST /api/v1/templates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var dto domain.CreateTemplateDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(GlobalResponse{
				Error:   "Invalid JSON body",
				Message: "Cuerpo de la petición inválido",
			})
			return
		}
		if dto.Name == "" || dto.DockerImage == "" || dto.BaseRamMB <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(GlobalResponse{
				Error:   "Missing or invalid required fields",
				Message: "Los campos name, docker_image y base_ram_mb (mayor a 0) son obligatorios",
			})
			return
		}
		id, err := db.CreateTemplate(r.Context(), dto)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(GlobalResponse{
				Error:   err.Error(),
				Message: "No se pudo crear la plantilla",
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(GlobalResponse{
			Data:    map[string]string{"id": id},
			Error:   "",
			Message: "Plantilla creada exitosamente",
		})
	})
	mux.HandleFunc("GET /api/v1/templates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		templates, err := db.GetAllTemplates(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(GlobalResponse{
				Error:   err.Error(),
				Message: "No se pudieron obtener las plantillas",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GlobalResponse{
			Data:    templates,
			Error:   "",
			Message: "Plantillas obtenidas exitosamente",
		})
	})
	mux.HandleFunc("POST /api/v1/test-docker", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		err := dockerManager.StartTestContainer(ctx)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, docker.ErrImagePullFailed) {
				status = http.StatusBadGateway
			}

			w.WriteHeader(status)
			json.NewEncoder(w).Encode(GlobalResponse{
				Data:    nil,
				Error:   err.Error(),
				Message: "No se pudo levantar el contenedor de prueba",
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(GlobalResponse{
			Data:    map[string]string{"url": "http://prueba.solv.local"},
			Error:   "",
			Message: "Contenedor de prueba desplegado exitosamente",
		})
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
