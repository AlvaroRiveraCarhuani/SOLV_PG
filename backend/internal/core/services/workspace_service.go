package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

var (
	ErrHostMemoryExhausted      = errors.New("host physical memory depleted: admission control denied request (HTTP 503)")
	ErrOOMKilledCooldownPenalty = errors.New("workspace reached 3 consecutive OOMKilled strikes: 5-minute cooldown penalty enforced")
)

type WorkspaceService struct {
	repo         domain.WorkspaceRepository
	docker       domain.WorkspaceOrchestrator
	hostMonitor  domain.HostMonitor
	subjectRepo  domain.SubjectRepository
	templateRepo domain.TemplateRepository
	db           *sqlx.DB
}

func NewWorkspaceService(repo domain.WorkspaceRepository, docker domain.WorkspaceOrchestrator, hostMonitor domain.HostMonitor) *WorkspaceService {
	return &WorkspaceService{
		repo:        repo,
		docker:      docker,
		hostMonitor: hostMonitor,
	}
}

// WithProvisioning extiende el servicio para resolver plantillas y bases de datos satélite en tiempo de ejecución
func (s *WorkspaceService) WithProvisioning(subjectRepo domain.SubjectRepository, templateRepo domain.TemplateRepository, db *sqlx.DB) *WorkspaceService {
	s.subjectRepo = subjectRepo
	s.templateRepo = templateRepo
	s.db = db
	return s
}

func getBaseDomain() string {
	d := strings.TrimSpace(os.Getenv("BASE_DOMAIN"))
	if d != "" {
		return d
	}
	return "solv.local"
}

func sanitizeIDForDB(id string) string {
	cleaned := strings.ToLower(strings.ReplaceAll(id, "-", ""))
	if len(cleaned) > 8 {
		return cleaned[:8]
	}
	return cleaned
}

func ensureSatellitePostgresDatabase(ctx context.Context, db *sqlx.DB, dbName string) error {
	for _, ch := range dbName {
		if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '_') {
			return fmt.Errorf("invalid characters in database name: %s", dbName)
		}
	}
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	if err := db.GetContext(ctx, &exists, checkQuery, dbName); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	if !exists {
		createQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
		if _, err := db.ExecContext(ctx, createQuery); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to create satellite database %s: %w", dbName, err)
			}
		}
		log.Printf("[Satellite DB] Created new isolated multi-tenant database: %s", dbName)
	}
	return nil
}

func getPostgresContainerHost() string {
	h := strings.TrimSpace(os.Getenv("POSTGRES_CONTAINER_HOST"))
	if h != "" {
		return h
	}
	if insideDocker := os.Getenv("INSIDE_DOCKER"); insideDocker == "true" {
		return "solv-postgres"
	}
	return "host.docker.internal"
}

func (s *WorkspaceService) resolveTemplate(ctx context.Context, subjectID string) *domain.Template {
	if s.subjectRepo == nil || s.templateRepo == nil || subjectID == "" {
		return nil
	}
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	subj, err := s.subjectRepo.GetByID(ctx, tenantID, subjectID)
	if err != nil || subj == nil || subj.TemplateID == nil || *subj.TemplateID == "" {
		return nil
	}
	tpl, err := s.templateRepo.GetTemplateByID(ctx, *subj.TemplateID)
	if err != nil || tpl == nil {
		return nil
	}
	return tpl
}

func (s *WorkspaceService) StartWorkspace(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	// 0. Resolución de plantilla institucional asociada a la materia
	template := s.resolveTemplate(ctx, subjectID)

	ramLimitMB := domain.DefaultBaseMemoryMB // 256
	imageName := domain.OpenVSCodeImage     // gitpod/openvscode-server:latest
	var templateIDStr *string
	if template != nil {
		templateIDStr = &template.ID
		if template.BaseRamMB >= 256 {
			ramLimitMB = int64(template.BaseRamMB)
		}
		if strings.TrimSpace(template.DockerImage) != "" {
			imageName = strings.TrimSpace(template.DockerImage)
		}
	}

	// 1. Verificación de Admisión del Host con la memoria real calculada
	if !s.hostMonitor.CanAllocateMemory(ramLimitMB) {
		return nil, ErrHostMemoryExhausted
	}

	// 2. Verificación de Idempotencia y Estado Previos
	existing, err := s.repo.GetByStudentAndSubject(ctx, studentID, subjectID)
	if err == nil && existing != nil {
		if existing.Status == domain.WorkspaceStatusRunning {
			return existing, nil
		}

		// Verificación de Castigo por OOMKilled (Sistema de 3 Strikes = 5 minutos de bloqueo)
		if existing.OOMStrikeCount >= domain.MaxOOMStrikes && existing.LastOOMKilledAt != nil {
			if time.Since(*existing.LastOOMKilledAt) < domain.OOMCooldownDuration {
				remaining := domain.OOMCooldownDuration - time.Since(*existing.LastOOMKilledAt)
				return nil, fmt.Errorf("%w (%d s remaining)", ErrOOMKilledCooldownPenalty, int(remaining.Seconds()))
			}
			_ = s.repo.ResetOOMStrikes(ctx, existing.ID)
		}

		return s.reactivateWorkspace(ctx, existing)
	}

	// 3. Generación de UUID opaco para el workspace_id y construcción de access_url
	workspaceID := uuid.NewString()
	baseDomain := getBaseDomain()
	scheme := "http"
	if strings.Contains(baseDomain, ".") && !strings.HasSuffix(baseDomain, ".local") {
		scheme = "https"
	}
	accessURL := fmt.Sprintf("%s://%s.%s", scheme, workspaceID, baseDomain)
	containerName := fmt.Sprintf("solv-workspace-%s", workspaceID)
	volumeName := fmt.Sprintf("solv_workspace_%s_%s", studentID, subjectID)
	networkName := "solv-traefik-net"

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}

	instance := &domain.WorkspaceInstance{
		ID:              workspaceID,
		TenantID:        tenantID,
		StudentID:       studentID,
		SubjectID:       subjectID,
		TemplateID:      templateIDStr,
		Status:          domain.WorkspaceStatusPending,
		AccessURL:       accessURL,
		MemoryLimitMB:   ramLimitMB,
		LastHeartbeatAt: time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 4. Crear registro pendiente en PostgreSQL mediante sqlx
	if err := s.repo.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create pending workspace: %w", err)
	}

	// 5. Asegurar la existencia del Volumen Nombrado (ADR-001) para la materia y estudiante
	if err := s.docker.EnsureVolumeExists(ctx, volumeName); err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to ensure named volume %s: %w", volumeName, err)
	}

	// 6. Asegurar la existencia de la red Docker compartida con Traefik deshabilitando ICC (enable_icc=false)
	if err := s.docker.EnsureICCDisabledNetworkExists(ctx, networkName); err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to ensure ICC-disabled docker network %s: %w", networkName, err)
	}

	// 7. Inyección de Dynamic Labels de Traefik v3 (Puerto 3000 para OpenVSCode Server)
	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", workspaceID):                      fmt.Sprintf("Host(`%s.%s`)", workspaceID, baseDomain),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", workspaceID): "3000",
	}

	envVars := []string{
		"DONT_PROMPT_WSL_INSTALL=true",
	}

	// 8. Aprovisionamiento Dinámico de Servicios Satélite Desacoplados (ADR-030)
	if template != nil && len(template.ServicesConfig.Services) > 0 && s.db != nil {
		for _, srv := range template.ServicesConfig.Services {
			if srv.Category == "database" && srv.Engine == "postgres" {
				cleanStd := sanitizeIDForDB(studentID)
				cleanSubj := sanitizeIDForDB(subjectID)
				dbName := fmt.Sprintf("solv_std_%s_%s", cleanStd, cleanSubj)
				if errDB := ensureSatellitePostgresDatabase(ctx, s.db, dbName); errDB != nil {
					log.Printf("[Satellite DB Warning] Failed to ensure database %s: %v", dbName, errDB)
				} else {
					pgHost := getPostgresContainerHost()
					dbURL := fmt.Sprintf("postgresql://postgres:postgres@%s:5432/%s?sslmode=disable", pgHost, dbName)
					envVars = append(envVars,
						fmt.Sprintf("DATABASE_URL=%s", dbURL),
						fmt.Sprintf("PGHOST=%s", pgHost),
						"PGPORT=5432",
						"PGUSER=postgres",
						"PGPASSWORD=postgres",
						fmt.Sprintf("PGDATABASE=%s", dbName),
					)
					log.Printf("[Satellite DB] Injected DATABASE_URL for workspace %s (db: %s)", workspaceID, dbName)
				}
			}
		}
	}

	// 9. Configuración del contenedor OpenVSCode Server
	config := domain.WorkspaceContainerConfig{
		Image:         imageName,
		ContainerName: containerName,
		VolumeName:    volumeName,
		MemoryLimitMB: ramLimitMB,
		NetworkName:   networkName,
		Labels:        labels,
		Env:           envVars,
	}

	// 10. Instanciación e inicio del contenedor OpenVSCode Server
	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to start openvscode-server container: %w", err)
	}

	// 11. Actualización del registro en PostgreSQL a estado 'running'
	if err := s.repo.UpdateContainerID(ctx, workspaceID, containerID); err != nil {
		return nil, fmt.Errorf("failed to update container_id: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusRunning); err != nil {
		return nil, fmt.Errorf("failed to update workspace status: %w", err)
	}

	// 12. Ejecución no bloqueante del setup_script si la plantilla lo define
	if template != nil && strings.TrimSpace(template.SetupScript) != "" {
		setupScript := template.SetupScript
		setupCmd := []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("cd /home/workspace && (%s) > /home/workspace/.solv_setup.log 2>&1", setupScript),
		}
		go func(cid, wid string) {
			time.Sleep(1 * time.Second)
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if errExec := s.docker.ExecuteCommandInBackground(bgCtx, cid, "/home/workspace", setupCmd); errExec != nil {
				log.Printf("[Setup Script Warning] Failed to execute setup_script in workspace %s: %v", wid, errExec)
			} else {
				log.Printf("[Setup Script] Successfully triggered setup_script for workspace %s", wid)
			}
		}(containerID, workspaceID)
	}

	instance.ContainerID = &containerID
	instance.Status = domain.WorkspaceStatusRunning
	return instance, nil
}

func (s *WorkspaceService) reactivateWorkspace(ctx context.Context, instance *domain.WorkspaceInstance) (*domain.WorkspaceInstance, error) {
	// Optimización ADR / Memory Governance: intentar despausar el contenedor congelado (instantáneo <50ms)
	if instance.ContainerID != nil && *instance.ContainerID != "" {
		if err := s.docker.UnpauseContainer(ctx, *instance.ContainerID); err == nil {
			_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusRunning)
			_ = s.repo.RecordHeartbeat(ctx, instance.ID)
			instance.Status = domain.WorkspaceStatusRunning
			return instance, nil
		}
	}

	template := s.resolveTemplate(ctx, instance.SubjectID)
	ramLimitMB := domain.DefaultBaseMemoryMB
	imageName := domain.OpenVSCodeImage
	if template != nil {
		if template.BaseRamMB >= 256 {
			ramLimitMB = int64(template.BaseRamMB)
		}
		if strings.TrimSpace(template.DockerImage) != "" {
			imageName = strings.TrimSpace(template.DockerImage)
		}
	}

	containerName := fmt.Sprintf("solv-workspace-%s", instance.ID)
	volumeName := fmt.Sprintf("solv_workspace_%s_%s", instance.StudentID, instance.SubjectID)
	networkName := "solv-traefik-net"
	baseDomain := getBaseDomain()

	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", instance.ID):                      fmt.Sprintf("Host(`%s.%s`)", instance.ID, baseDomain),
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", instance.ID): "3000",
	}

	envVars := []string{
		"DONT_PROMPT_WSL_INSTALL=true",
	}

	// Reinyección de credenciales satélite si aplica
	if template != nil && len(template.ServicesConfig.Services) > 0 && s.db != nil {
		for _, srv := range template.ServicesConfig.Services {
			if srv.Category == "database" && srv.Engine == "postgres" {
				cleanStd := sanitizeIDForDB(instance.StudentID)
				cleanSubj := sanitizeIDForDB(instance.SubjectID)
				dbName := fmt.Sprintf("solv_std_%s_%s", cleanStd, cleanSubj)
				_ = ensureSatellitePostgresDatabase(ctx, s.db, dbName)
				pgHost := getPostgresContainerHost()
				dbURL := fmt.Sprintf("postgresql://postgres:postgres@%s:5432/%s?sslmode=disable", pgHost, dbName)
				envVars = append(envVars,
					fmt.Sprintf("DATABASE_URL=%s", dbURL),
					fmt.Sprintf("PGHOST=%s", pgHost),
					"PGPORT=5432",
					"PGUSER=postgres",
					"PGPASSWORD=postgres",
					fmt.Sprintf("PGDATABASE=%s", dbName),
				)
			}
		}
	}

	config := domain.WorkspaceContainerConfig{
		Image:         imageName,
		ContainerName: containerName,
		VolumeName:    volumeName,
		MemoryLimitMB: ramLimitMB,
		NetworkName:   networkName,
		Labels:        labels,
		Env:           envVars,
	}

	containerID, err := s.docker.StartWorkspaceContainer(ctx, config)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusFailed)
		return nil, fmt.Errorf("failed to reactivate workspace container: %w", err)
	}

	_ = s.repo.UpdateContainerID(ctx, instance.ID, containerID)
	_ = s.repo.UpdateStatus(ctx, instance.ID, domain.WorkspaceStatusRunning)
	_ = s.repo.UpdateMemoryLimit(ctx, instance.ID, ramLimitMB)
	_ = s.repo.RecordHeartbeat(ctx, instance.ID)

	instance.ContainerID = &containerID
	instance.Status = domain.WorkspaceStatusRunning
	instance.MemoryLimitMB = ramLimitMB
	return instance, nil
}

func (s *WorkspaceService) RecordHeartbeat(ctx context.Context, workspaceID string) error {
	return s.repo.RecordHeartbeat(ctx, workspaceID)
}

func (s *WorkspaceService) RestartWorkspace(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		_ = s.docker.StopAndRemoveContainer(ctx, *ws.ContainerID)
	}

	_ = s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusPending)
	return s.reactivateWorkspace(ctx, ws)
}

func (s *WorkspaceService) TerminateWorkspace(ctx context.Context, workspaceID string) error {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.StopAndRemoveContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("Warning: failed to stop/remove container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusFailed); err != nil {
		return fmt.Errorf("failed to update workspace status to failed: %w", err)
	}

	return nil
}

func (s *WorkspaceService) HibernateWorkspace(ctx context.Context, workspaceID string) error {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.PauseContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("Warning: failed to pause container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusHibernated); err != nil {
		return fmt.Errorf("failed to update workspace status to hibernated: %w", err)
	}

	return nil
}

func (s *WorkspaceService) GetWorkspaceByID(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	return s.repo.GetByID(ctx, workspaceID)
}

func (s *WorkspaceService) PauseWorkspace(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	if ws.ContainerID != nil && *ws.ContainerID != "" {
		if err := s.docker.PauseContainer(ctx, *ws.ContainerID); err != nil {
			log.Printf("[WorkspaceService] Warning: error pausing container %s: %v", *ws.ContainerID, err)
		}
	}

	if err := s.repo.UpdateStatus(ctx, workspaceID, domain.WorkspaceStatusHibernated); err != nil {
		return nil, fmt.Errorf("failed to update workspace status to hibernated: %w", err)
	}

	ws.Status = domain.WorkspaceStatusHibernated
	return ws, nil
}

func (s *WorkspaceService) GetSemgrepAudit(ctx context.Context, workspaceID string) (*domain.WorkspaceInstance, error) {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}
	return ws, nil
}

