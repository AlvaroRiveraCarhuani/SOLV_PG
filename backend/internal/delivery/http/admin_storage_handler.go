package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// AdminStorageHandler expone el inventario de volúmenes sin contenedor y la
// liberación confirmada desde Configuración → Servidor. El panel muestra el
// tamaño y el motivo de cada decisión; el borrado exige la frase exacta y deja
// una entrada de auditoría por volumen.
type AdminStorageHandler struct {
	diskService *services.DiskOrphanService
}

func NewAdminStorageHandler(diskService *services.DiskOrphanService) *AdminStorageHandler {
	return &AdminStorageHandler{diskService: diskService}
}

// GetOrphanVolumes responde qué volúmenes colgantes se pueden liberar y cuáles
// se conservan con su motivo.
func (h *AdminStorageHandler) GetOrphanVolumes(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.diskService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "El inventario de almacenamiento no está disponible en esta instancia")
		return
	}

	plan, err := h.diskService.Plan(r.Context())
	if err != nil {
		sendDiskOrphanError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, plan, "Inventario de volúmenes sin contenedor")
}

// PruneOrphanVolumes libera los volúmenes elegidos tras la confirmación.
func (h *AdminStorageHandler) PruneOrphanVolumes(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.diskService == nil {
		SendError(w, http.StatusServiceUnavailable, "service_unavailable", "El inventario de almacenamiento no está disponible en esta instancia")
		return
	}

	var req struct {
		VolumeNames        []string `json:"volume_names"`
		ConfirmationPhrase string   `json:"confirmation_phrase"`
		Reason             string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid_payload", "payload JSON inválido")
		return
	}

	tenantID := getTenantFromCtx(r)
	actorID := requestUserID(r)

	result, err := h.diskService.Prune(r.Context(), tenantID, actorID, domain.OrphanVolumePruneRequest{
		VolumeNames:        req.VolumeNames,
		ConfirmationPhrase: req.ConfirmationPhrase,
		Reason:             req.Reason,
	})
	if err != nil {
		sendDiskOrphanError(w, err)
		return
	}

	SendJSON(w, http.StatusOK, result, "Volúmenes huérfanos liberados")
}

func sendDiskOrphanError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidConfirmationPhrase):
		SendError(w, http.StatusUnprocessableEntity, "invalid_confirmation_phrase", "La frase de confirmación no coincide exactamente")
	case errors.Is(err, services.ErrOrphanVolumeSelectionEmpty):
		SendError(w, http.StatusBadRequest, "empty_selection", "Selecciona al menos un volumen para liberar")
	case errors.Is(err, services.ErrOrphanVolumeSelectionTooLarge):
		SendError(w, http.StatusUnprocessableEntity, "selection_too_large", "Selecciona menos volúmenes por corrida")
	case errors.Is(err, services.ErrVolumeInventoryUnavailable):
		SendError(w, http.StatusServiceUnavailable, "inventory_unavailable", "No se pudo leer el daemon o los entornos registrados; no se borró nada")
	default:
		SendError(w, http.StatusInternalServerError, err.Error(), "Error al liberar los volúmenes huérfanos")
	}
}
