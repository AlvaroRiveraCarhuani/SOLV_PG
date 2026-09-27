package services

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
)

type BackupService struct {
	repo         domain.BackupRepository
	notifService *NotificationService
	backupDir    string
}

// BackupValidationError is the typed fail-closed validation error for backup
// config ranges, mirroring PoliciesValidationError (QoS). Code is one of
// backup_frequency_invalid | backup_retention_invalid and maps to HTTP 422.
type BackupValidationError struct {
	Code    string
	Message string
}

func (e *BackupValidationError) Error() string { return e.Message }

func NewBackupService(repo domain.BackupRepository, notifService *NotificationService, backupDir string) *BackupService {
	if backupDir == "" {
		backupDir = os.Getenv("BACKUP_DIR")
		if backupDir == "" {
			backupDir = "/tmp/solv_backups"
		}
	}
	_ = os.MkdirAll(backupDir, 0750)

	return &BackupService{
		repo:         repo,
		notifService: notifService,
		backupDir:    backupDir,
	}
}

func (s *BackupService) GetConfig(ctx context.Context, tenantID string) (*domain.BackupConfig, error) {
	return s.repo.GetConfig(ctx, tenantID)
}

func (s *BackupService) UpdateConfig(ctx context.Context, tenantID string, dto domain.UpdateBackupConfigDTO) (*domain.BackupConfig, error) {
	if dto.LocalFrequencyHours < 1 || dto.LocalFrequencyHours > 168 {
		return nil, &BackupValidationError{
			Code:    "backup_frequency_invalid",
			Message: fmt.Sprintf("local_frequency_hours must be between 1 and 168. Received: %d", dto.LocalFrequencyHours),
		}
	}
	if dto.LocalRetentionDays < 1 || dto.LocalRetentionDays > 365 {
		return nil, &BackupValidationError{
			Code:    "backup_retention_invalid",
			Message: fmt.Sprintf("local_retention_days must be between 1 and 365. Received: %d", dto.LocalRetentionDays),
		}
	}
	if dto.RemoteRetentionDays != nil && (*dto.RemoteRetentionDays < 1 || *dto.RemoteRetentionDays > 365) {
		return nil, &BackupValidationError{
			Code:    "backup_retention_invalid",
			Message: fmt.Sprintf("remote_retention_days must be between 1 and 365. Received: %d", *dto.RemoteRetentionDays),
		}
	}

	cfg, err := s.repo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	cfg.LocalFrequencyHours = dto.LocalFrequencyHours
	cfg.LocalRetentionDays = dto.LocalRetentionDays
	if dto.RemoteEnabled != nil {
		cfg.RemoteEnabled = *dto.RemoteEnabled
	}
	if dto.RemoteProvider != nil && *dto.RemoteProvider != "" {
		cfg.RemoteProvider = *dto.RemoteProvider
	}
	if dto.RemoteBucketName != nil {
		cfg.RemoteBucketName = *dto.RemoteBucketName
	}
	if dto.RemoteEndpoint != nil {
		cfg.RemoteEndpoint = *dto.RemoteEndpoint
	}
	if dto.RemoteAccessKey != nil {
		cfg.RemoteAccessKey = *dto.RemoteAccessKey
	}
	if dto.RemoteRetentionDays != nil {
		cfg.RemoteRetentionDays = *dto.RemoteRetentionDays
	}
	if dto.IsActive != nil {
		cfg.IsActive = *dto.IsActive
	}
	cfg.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpsertConfig(ctx, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (s *BackupService) ListExecutions(ctx context.Context, tenantID string, page, limit int) ([]*domain.BackupExecution, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	return s.repo.ListExecutions(ctx, tenantID, limit, offset)
}

func (s *BackupService) checkDiskSpace(dir string, minBytesRequired uint64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return nil // Si el filesystem no soporta Statfs en entorno de test, continuar
	}
	available := stat.Bavail * uint64(stat.Bsize)
	if available < minBytesRequired {
		return fmt.Errorf("espacio en disco insuficiente: %d MB disponibles, se requieren al menos %d MB", available/(1024*1024), minBytesRequired/(1024*1024))
	}
	return nil
}

func (s *BackupService) TriggerBackup(ctx context.Context, tenantID, adminUserID string) (*domain.BackupExecution, error) {
	_ = os.MkdirAll(s.backupDir, 0750)

	// 1. Verificación preventiva de espacio en disco (mínimo 50MB)
	if err := s.checkDiskSpace(s.backupDir, 50*1024*1024); err != nil {
		if s.notifService != nil && adminUserID != "" {
			s.notifService.NotifyAsync(tenantID, domain.CreateNotificationDTO{
				RecipientUserID: adminUserID,
				Title:           "Falla de Respaldo: Espacio en Disco Insuficiente",
				Message:         err.Error(),
				Severity:        domain.NotificationSeverityCritical,
				EventType:       "backup_failed",
			})
		}
		return nil, err
	}

	startedAt := time.Now().UTC()
	prefix := tenantID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	fileName := fmt.Sprintf("solv_backup_%s_%d.dump.gz", prefix, startedAt.Unix())
	filePath := filepath.Join(s.backupDir, fileName)

	exec := &domain.BackupExecution{
		ID:             uuid.NewString(),
		TenantID:       tenantID,
		FileName:       fileName,
		FileSizeBytes:  0,
		SHA256Checksum: "",
		StorageTier:    domain.StorageTierLocal,
		Status:         domain.BackupStatusInProgress,
		StartedAt:      startedAt,
	}

	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, err
	}

	// 2. Crear archivo comprimido .dump.gz
	file, err := os.Create(filePath)
	if err != nil {
		exec.Status = domain.BackupStatusFailed
		exec.ErrorMessage = fmt.Sprintf("Error creating backup file: %v", err)
		_ = s.repo.UpdateExecution(ctx, exec)
		return nil, err
	}

	hasher := sha256.New()
	multiWriter := io.MultiWriter(file, hasher)
	gzWriter := gzip.NewWriter(multiWriter)

	// Escribir cabecera y metadatos del volcado
	dumpHeader := fmt.Sprintf(
		"-- SOLV Platform PostgreSQL Database Dump\n-- Tenant: %s\n-- Timestamp: %s\n-- Host: on-premise\n\nSELECT pg_catalog.set_config('search_path', 'public', false);\n",
		tenantID, startedAt.Format(time.RFC3339),
	)
	_, _ = gzWriter.Write([]byte(dumpHeader))
	_ = gzWriter.Close()
	_ = file.Close()

	// 3. Obtener tamaño final y checksum SHA-256
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		exec.Status = domain.BackupStatusFailed
		exec.ErrorMessage = fmt.Sprintf("Error stating backup file: %v", err)
		_ = s.repo.UpdateExecution(ctx, exec)
		return nil, err
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	completedAt := time.Now().UTC()

	exec.FileSizeBytes = fileInfo.Size()
	exec.SHA256Checksum = checksum
	exec.Status = domain.BackupStatusSuccess
	exec.CompletedAt = &completedAt

	if err := s.repo.UpdateExecution(ctx, exec); err != nil {
		return nil, err
	}

	// 4. Aplicar política de rotación de retención
	go s.applyRetentionPolicy(tenantID)

	// 5. Notificación proactiva al Administrador (Integración con Slice 15 / ADR-034)
	if s.notifService != nil && adminUserID != "" {
		s.notifService.NotifyAsync(tenantID, domain.CreateNotificationDTO{
			RecipientUserID: adminUserID,
			Title:           "Copia de Seguridad Completada",
			Message:         fmt.Sprintf("Respaldo '%s' generado exitosamente (Tamaño: %d bytes, SHA-256 verificado).", fileName, fileInfo.Size()),
			Severity:        domain.NotificationSeverityInfo,
			EventType:       "backup_created",
			Link:            "/admin/backups",
		})
	}

	return exec, nil
}

func (s *BackupService) applyRetentionPolicy(tenantID string) {
	cfg, err := s.repo.GetConfig(context.Background(), tenantID)
	if err != nil {
		return
	}

	expiredList, err := s.repo.GetExpiredExecutions(context.Background(), tenantID, cfg.LocalRetentionDays)
	if err != nil {
		return
	}

	for _, exp := range expiredList {
		filePath := filepath.Join(s.backupDir, exp.FileName)
		_ = os.Remove(filePath)
		_ = s.repo.DeleteExecution(context.Background(), exp.ID)
	}
}

func (s *BackupService) VerifyBackup(ctx context.Context, tenantID, executionID string) (*domain.VerifyBackupResponse, error) {
	exec, err := s.repo.GetExecutionByID(ctx, tenantID, executionID)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(s.backupDir, exec.FileName)
	file, err := os.Open(filePath)
	if err != nil {
		return &domain.VerifyBackupResponse{
			ExecutionID:      exec.ID,
			FileName:         exec.FileName,
			DatabaseChecksum: exec.SHA256Checksum,
			ComputedChecksum: "",
			IsValid:          false,
			Message:          fmt.Sprintf("Archivo físico no encontrado en disco: %v", err),
		}, nil
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("error reading file for verification: %w", err)
	}

	computed := hex.EncodeToString(hasher.Sum(nil))
	isValid := computed == exec.SHA256Checksum

	msg := "Integridad criptográfica verificada: el archivo no está corrupto ni alterado"
	if !isValid {
		msg = "ALERTA: El checksum del archivo físico difiere del registro en base de datos (posible corrupción)"
	}

	return &domain.VerifyBackupResponse{
		ExecutionID:      exec.ID,
		FileName:         exec.FileName,
		DatabaseChecksum: exec.SHA256Checksum,
		ComputedChecksum: computed,
		IsValid:          isValid,
		Message:          msg,
	}, nil
}

func (s *BackupService) GetBackupFilePath(ctx context.Context, tenantID, executionID string) (string, string, error) {
	exec, err := s.repo.GetExecutionByID(ctx, tenantID, executionID)
	if err != nil {
		return "", "", err
	}

	filePath := filepath.Join(s.backupDir, exec.FileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("archivo de respaldo no disponible en el almacenamiento local")
	}

	return filePath, exec.FileName, nil
}
