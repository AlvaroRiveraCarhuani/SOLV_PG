package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"solv-backend/internal/core/domain"
)

const serverPoliciesConfigKey = "server_policies"

// HostTotalRAMProbe mide la RAM total del host en MB. Se inyecta desde main con
// la medición real (system.HostTotalRAMMB). Sin probe, el techo de concurrencia
// queda en el default histórico.
type HostTotalRAMProbe func() (int64, error)

// ServerPoliciesService gestiona las políticas QoS del servidor persistidas en
// tenants.config. Es la fuente de verdad del worker QoS (que las recarga en
// cada ciclo) y del provisionamiento de environments nuevos (RAM).
type ServerPoliciesService struct {
	tenantRepo   domain.TenantRepository
	hostTotalRAM HostTotalRAMProbe
}

func NewServerPoliciesService(tenantRepo domain.TenantRepository) *ServerPoliciesService {
	return &ServerPoliciesService{tenantRepo: tenantRepo}
}

// WithHostTotalRAM habilita el cálculo del techo recomendado sobre la RAM medida.
func (s *ServerPoliciesService) WithHostTotalRAM(probe HostTotalRAMProbe) *ServerPoliciesService {
	s.hostTotalRAM = probe
	return s
}

// Get devuelve las políticas vigentes. Si el tenant nunca las guardó, el techo de
// concurrencia se deriva de la RAM medida en este host: un valor fijo no puede
// servir a la vez para un servidor de laboratorio y para uno de facultad.
func (s *ServerPoliciesService) Get(ctx context.Context, tenantID string) (*domain.ServerPolicies, error) {
	policies, persisted, err := s.lookup(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if persisted {
		return policies, nil
	}

	recommendation := s.Recommendation(*policies)
	if recommendation.CapacityBasis == domain.CapacityBasisHostRAM && recommendation.RecommendedMaxContainers > 0 {
		policies.MaxContainers = recommendation.RecommendedMaxContainers
	}
	return policies, nil
}

// Recommendation explica de dónde sale el techo recomendado con las políticas
// dadas. No modifica nada: es lo que el panel muestra junto al valor guardado.
func (s *ServerPoliciesService) Recommendation(policies domain.ServerPolicies) domain.CapacityRecommendation {
	var totalMB int64
	if s.hostTotalRAM != nil {
		if measured, err := s.hostTotalRAM(); err == nil {
			totalMB = measured
		}
	}
	return domain.BuildCapacityRecommendation(totalMB, policies.RAMLimitMB, policies.HostSafetyCeilingPct)
}

// lookup lee las políticas persistidas y distingue "guardadas" de "defaults": sin
// esa distinción no se puede saber si el techo vigente es una decisión del
// administrador o la ausencia de configuración.
func (s *ServerPoliciesService) lookup(ctx context.Context, tenantID string) (*domain.ServerPolicies, bool, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return nil, false, fmt.Errorf("tenant not found")
	}

	defaults := domain.DefaultServerPolicies()
	if len(tenant.Config) == 0 {
		return &defaults, false, nil
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(tenant.Config, &configMap); err != nil {
		return &defaults, false, nil
	}

	raw, ok := configMap[serverPoliciesConfigKey]
	if !ok {
		return &defaults, false, nil
	}

	policies := defaults
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return &defaults, false, nil
	}
	if err := json.Unmarshal(rawBytes, &policies); err != nil {
		return &defaults, false, nil
	}
	if policies.RAMLimitMB <= 0 || policies.InactivityMinutes <= 0 || policies.MaxContainers <= 0 {
		return &defaults, false, nil
	}
	if policies.HostSafetyCeilingPct <= 0 {
		policies.HostSafetyCeilingPct = 85
	}
	return &policies, true, nil
}

// Update aplica cambios parciales con validación estricta de rangos.
func (s *ServerPoliciesService) Update(ctx context.Context, tenantID string, dto domain.UpdateServerPoliciesDTO) (*domain.ServerPolicies, error) {
	current, persisted, err := s.lookup(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	explicitCap := dto.MaxContainers != nil

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
		if *dto.MaxContainers < 1 || *dto.MaxContainers > domain.MaxAllowedContainers {
			return nil, &PoliciesValidationError{
				Code:    "max_containers_invalid",
				Message: fmt.Sprintf("max_containers debe estar entre 1 y %d. Recibido: %d", domain.MaxAllowedContainers, *dto.MaxContainers),
			}
		}
		current.MaxContainers = *dto.MaxContainers
	}
	if dto.HostSafetyCeilingPct != nil {
		if !containsInt(domain.AllowedHostSafetyCeilingsPct, *dto.HostSafetyCeilingPct) {
			return nil, &PoliciesValidationError{
				Code:    "host_ceiling_invalid",
				Message: fmt.Sprintf("host_safety_ceiling_pct debe ser uno de: %s. Recibido: %d", allowedList(domain.AllowedHostSafetyCeilingsPct), *dto.HostSafetyCeilingPct),
			}
		}
		current.HostSafetyCeilingPct = *dto.HostSafetyCeilingPct
	}

	// Un techo derivado acompaña al techo de seguridad que se acaba de guardar:
	// si no, bajar el techo de 85% a 80% dejaría un max_containers calculado con
	// el valor viejo.
	if !persisted && !explicitCap {
		if recommendation := s.Recommendation(*current); recommendation.CapacityBasis == domain.CapacityBasisHostRAM && recommendation.RecommendedMaxContainers > 0 {
			current.MaxContainers = recommendation.RecommendedMaxContainers
		}
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
