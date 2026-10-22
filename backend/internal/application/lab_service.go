package application

import (
	"context"
	"fmt"
	"os"
	"strings"

	"solv-backend/internal/domain"
)

// publicBaseDomain devuelve el dominio público configurado. Un valor .local se
// descarta: el acceso local se resuelve por localhost.
func publicBaseDomain() string {
	d := strings.TrimSpace(os.Getenv("BASE_DOMAIN"))
	if d == "" || strings.HasSuffix(strings.ToLower(d), ".local") {
		return "localhost"
	}
	return d
}

type DatabasePort interface {
	domain.UserRepository
	domain.TemplateRepository
	domain.InstanceRepository
}

type LabService struct {
	db     DatabasePort
	docker domain.DockerService
}

func NewLabService(db DatabasePort, docker domain.DockerService) *LabService {
	return &LabService{
		db:     db,
		docker: docker,
	}
}

func (s *LabService) StartLab(ctx context.Context, userID, templateID string) (domain.CreateInstanceDTO, error) {
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return domain.CreateInstanceDTO{}, fmt.Errorf("failed to get user: %w", err)
	}

	template, err := s.db.GetTemplateByID(ctx, templateID)
	if err != nil {
		return domain.CreateInstanceDTO{}, fmt.Errorf("failed to get template: %w", err)
	}

	prefix := strings.Split(user.Email, "@")[0]
	prefix = strings.ReplaceAll(prefix, ".", "_")
	prefix = strings.ToLower(prefix)

	cleanTemplateName := strings.ReplaceAll(strings.ToLower(template.Name), " ", "")
	containerName := fmt.Sprintf("solv_%s_%s", cleanTemplateName, prefix)
	traefikURL := fmt.Sprintf("%s.%s", prefix, publicBaseDomain())

	dto := domain.CreateInstanceDTO{
		UserID:        userID,
		TemplateID:    templateID,
		ContainerName: containerName,
		TraefikURL:    traefikURL,
		Status:        "pending",
	}

	instanceID, err := s.db.CreateInstance(ctx, dto)
	if err != nil {
		return domain.CreateInstanceDTO{}, fmt.Errorf("failed to create instance record: %w", err)
	}

	err = s.docker.StartContainer(ctx, template.DockerImage, containerName, traefikURL)
	if err != nil {
		rollbackErr := s.db.UpdateInstanceStatus(ctx, instanceID, "failed")
		if rollbackErr != nil {
			return domain.CreateInstanceDTO{}, fmt.Errorf("CRITICAL: failed to start docker container (%w) AND failed to update instance status to failed (%w)", err, rollbackErr)
		}
		return domain.CreateInstanceDTO{}, fmt.Errorf("failed to start docker container: %w", err)
	}

	err = s.db.UpdateInstanceStatus(ctx, instanceID, "running")
	if err != nil {
		return domain.CreateInstanceDTO{}, fmt.Errorf("container started but failed to update status to running: %w", err)
	}
	dto.Status = "running"

	return dto, nil
}
