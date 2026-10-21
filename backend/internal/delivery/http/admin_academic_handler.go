package httpdelivery

import (
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type AdminAcademicHandler struct {
	periodService       *services.AcademicPeriodService
	maintenanceService  *services.MaintenanceService
	govService          *services.AdminGovernanceService
	imageService        *services.ImageVerificationService
	auditLogRepo        domain.AuditLogRepository
	cacheService        *services.TemplateCacheService
	notificationService *services.NotificationService
}

func NewAdminAcademicHandler(
	periodService *services.AcademicPeriodService,
	maintenanceService *services.MaintenanceService,
	govService *services.AdminGovernanceService,
) *AdminAcademicHandler {
	return &AdminAcademicHandler{
		periodService:      periodService,
		maintenanceService: maintenanceService,
		govService:         govService,
	}
}

func (h *AdminAcademicHandler) WithImageService(imageService *services.ImageVerificationService) *AdminAcademicHandler {
	h.imageService = imageService
	return h
}

func (h *AdminAcademicHandler) WithAuditLogRepo(auditLogRepo domain.AuditLogRepository) *AdminAcademicHandler {
	h.auditLogRepo = auditLogRepo
	return h
}

func (h *AdminAcademicHandler) WithCacheService(cacheService *services.TemplateCacheService) *AdminAcademicHandler {
	h.cacheService = cacheService
	return h
}

func (h *AdminAcademicHandler) WithNotificationService(notificationService *services.NotificationService) *AdminAcademicHandler {
	h.notificationService = notificationService
	return h
}
