package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"solv-backend/internal/core/domain"
)

const serverPoliciesConfigKey = "server_policies"

// ServerPoliciesService gestiona las políticas QoS del servidor persistidas en
// tenants.config. Es la fuente de verdad del worker QoS (que las recarga en
// cada ciclo) y del provisionamiento de workspaces nuevos (RAM).
type ServerPoliciesService struct {
	tenantRepo domain.TenantRepository
}

func NewServerPoliciesService(tenantRepo domain.TenantRepository) *ServerPoliciesService {
	return &ServerPoliciesService{tenantRepo: tenantRepo}
}

// Get devuelve las políticas vigentes o los defaults si nunca se configuraron.
func (s *ServerPoliciesService) Get(ctx context.Context, tenantID string) (*domain.ServerPolicies, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	defaults := domain.DefaultServerPolicies()
	if len(tenant.Config) == 0 {
		return &defaults, nil
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(tenant.Config, &configMap); err != nil {
		return &defaults, nil
	}

	raw, ok := configMap[serverPoliciesConfigKey]
	if !ok {
		return &defaults, nil
	}

	policies := defaults
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return &defaults, nil
	}
	if err := json.Unmarshal(rawBytes, &policies); err != nil {
		return &defaults, nil
	}
	if policies.RAMLimitMB <= 0 || policies.InactivityMinutes <= 0 || policies.MaxContainers <= 0 {
		return &defaults, nil
	}
	return &policies, nil
}

// Update aplica cambios parciales con validación estricta de rangos.
func (s *ServerPoliciesService) Update(ctx context.Context, tenantID string, dto domain.UpdateServerPoliciesDTO) (*domain.ServerPolicies, error) {
	current, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if dto.RAMLimitMB != nil {
		if !containsInt(domain.AllowedRAMLimitsMB, *dto.RAMLimitMB) {
			return nil, &PoliciesValidationError{
				Code:    "ram_limit_invalid",
				Message: fmt.Sprintf("ram_limit_mb debe ser uno de: %s MB. Recibido: %d", allowedList(domain.AllowedRAMLimitsMB), *dto.RAMLimitMB),
			}
		}
		current.RAMLimitMB = *dto.RAMLimitMB
	}
	if dto.InactivityMinutes != nil {
		if !containsInt(domain.AllowedInactivityMinutes, *dto.InactivityMinutes) {
			return nil, &PoliciesValidationError{
				Code:    "inactivity_invalid",
				Message: fmt.Sprintf("inactivity_minutes debe ser uno de: %s min. Recibido: %d", allowedList(domain.AllowedInactivityMinutes), *dto.InactivityMinutes),
			}
		}
		current.InactivityMinutes = *dto.InactivityMinutes
	}
	if dto.MaxContainers != nil {
		if *dto.MaxContainers < 1 || *dto.MaxContainers > 500 {
			return nil, &PoliciesValidationError{
				Code:    "max_containers_invalid",
				Message: fmt.Sprintf("max_containers debe estar entre 1 y 500. Recibido: %d", *dto.MaxContainers),
			}
		}
		current.MaxContainers = *dto.MaxContainers
	}

	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.persist(ctx, tenantID, *current); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *ServerPoliciesService) persist(ctx context.Context, tenantID string, policies domain.ServerPolicies) error {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return fmt.Errorf("tenant not found")
	}

	var configMap map[string]interface{}
	if len(tenant.Config) > 0 {
		if err := json.Unmarshal(tenant.Config, &configMap); err != nil {
			configMap = make(map[string]interface{})
		}
	}
	if configMap == nil {
		configMap = make(map[string]interface{})
	}

	configMap[serverPoliciesConfigKey] = policies
	newConfig, err := json.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("error serializing server policies: %w", err)
	}

	return s.tenantRepo.UpdateConfig(ctx, tenantID, newConfig)
}

type PoliciesValidationError struct {
	Code    string
	Message string
}

func (e *PoliciesValidationError) Error() string { return e.Message }

func containsInt(list []int, v int) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func allowedList(list []int) string {
	out := ""
	for i, item := range list {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%d", item)
	}
	return out
}
