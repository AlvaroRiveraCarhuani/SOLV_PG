package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

type WorkspaceService struct {
	repo            domain.WorkspaceRepository
	docker          domain.WorkspaceOrchestrator
	hostMonitor     domain.HostMonitor
	subjectRepo     domain.SubjectRepository
	templateRepo    domain.TemplateRepository
	db              *sqlx.DB
	policiesService *ServerPoliciesService
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

// SetPoliciesService inyecta las políticas QoS del tenant (submódulo 14.6):
// la RAM de la política es el default por alumno y max_containers el techo
// de concurrencia en la admisión. Sin inyección rigen los defaults del dominio.
func (s *WorkspaceService) SetPoliciesService(policies *ServerPoliciesService) *WorkspaceService {
	s.policiesService = policies
	return s
}

// effectivePolicies resuelve las políticas vigentes del tenant con fallback a
// defaults ante cualquier fallo (fail-open: una política nunca voltea clases;
// la protección dura del host vive en el admission control físico).
func (s *WorkspaceService) effectivePolicies(ctx context.Context, tenantID string) domain.ServerPolicies {
	defaults := domain.DefaultServerPolicies()
	if s.policiesService == nil {
		return defaults
	}
	policies, err := s.policiesService.Get(ctx, tenantID)
	if err != nil || policies == nil {
		return defaults
	}
	return *policies
}

func getBaseDomain() string {
	d := strings.TrimSpace(os.Getenv("BASE_DOMAIN"))
	if d != "" {
		return d
	}
	return "solv.local"
}

// hostTotalMB lee la RAM física total del host con fallback a 8 GB cuando
// gopsutil no responde (mismo contrato que el resto de los workers).
func hostTotalMB() int {
	if vm, err := mem.VirtualMemory(); err == nil && vm.Total > 0 {
		return int(vm.Total / (1024 * 1024))
	}
	return 8192
}

func sanitizeIDForDB(id string) string {
	cleaned := strings.ToLower(strings.ReplaceAll(id, "-", ""))
	if len(cleaned) > 8 {
		return cleaned[:8]
	}
	return cleaned
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

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	policies := s.effectivePolicies(ctx, tenantID)

	// 0.1 Techo de concurrencia del tenant (max_containers): hard gate.
	// Solo cuenta workspaces nuevos; reactivaciones de existentes no consumen cupo.
	if active, err := s.repo.GetActiveWorkspaces(ctx); err != nil {
		log.Printf("[Admission] WARNING: no se pudo contar workspaces activos, admitiendo por fail-open: %v", err)
	} else if len(active) >= policies.MaxContainers {
		return nil, fmt.Errorf("%w: tenant %s tiene %d activos (techo %d)", ErrMaxContainersReached, tenantID, len(active), policies.MaxContainers)
	}

	// 0.2 RAM efectiva: la política es el default por alumno; la plantilla
	// prevalece cuando declara un requisito (BaseRamMB >= 256) y el exceso
	// sobre la política queda registrado para visibilidad del admin.
	ramLimitMB := int64(policies.RAMLimitMB)
	imageName := domain.OpenVSCodeImage // gitpod/openvscode-server:latest
	var templateIDStr *string
	if template != nil {
		templateIDStr = &template.ID
		if template.BaseRamMB >= 256 {
			ramLimitMB = int64(template.BaseRamMB)
			if int(ramLimitMB) > policies.RAMLimitMB {
				log.Printf("[Admission PolicyOverride] plantilla %s requiere %d MB sobre la política del tenant (%d MB): prevalece la plantilla",
					template.ID, ramLimitMB, policies.RAMLimitMB)
			}
		}
		if err := domain.ValidateRamAgainstHost(int(ramLimitMB), domain.CalculateHostMaxAllowedRAM(hostTotalMB())); err != nil {
			return nil, fmt.Errorf("solicitud rechazada: la memoria de la plantilla (%d MB) excede la capacidad estructural del host", ramLimitMB)
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
