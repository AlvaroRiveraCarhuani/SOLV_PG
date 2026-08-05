package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresLabTemplateRepository struct {
	db *sqlx.DB
}

func NewPostgresLabTemplateRepository(db *sqlx.DB) domain.LabTemplateRepository {
	return &PostgresLabTemplateRepository{db: db}
}

type labTemplateProfileRow struct {
	SignatureHash   string    `db:"signature_hash"`
	Name            string    `db:"name"`
	BaseImage       string    `db:"base_image"`
	SetupScript     string    `db:"setup_script"`
	ResourceProfile []byte    `db:"resource_profile"`
	TenantID        string    `db:"tenant_id"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func (r *PostgresLabTemplateRepository) GetBySignatureHash(ctx context.Context, signatureHash string) (*domain.LabTemplate, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
	SELECT signature_hash, name, base_image, setup_script, resource_profile, tenant_id, created_at, updated_at
	FROM lab_template_profiles
	WHERE signature_hash = $1 AND tenant_id = $2;`

	var row labTemplateProfileRow
	if err := r.db.GetContext(ctx, &row, query, signatureHash, tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get lab template profile by hash: %w", err)
	}

	var profile domain.ResourceProfile
	if len(row.ResourceProfile) > 0 {
		if err := json.Unmarshal(row.ResourceProfile, &profile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource profile json: %w", err)
		}
	}

	return &domain.LabTemplate{
		ID:              row.SignatureHash,
		Name:            row.Name,
		BaseImage:       row.BaseImage,
		SetupScript:     row.SetupScript,
		SignatureHash:   row.SignatureHash,
		ResourceProfile: profile,
		TenantID:        row.TenantID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *PostgresLabTemplateRepository) CreateOrUpdateProfile(ctx context.Context, template *domain.LabTemplate) error {
	profileJSON, err := json.Marshal(template.ResourceProfile)
	if err != nil {
		return fmt.Errorf("failed to marshal resource profile: %w", err)
	}

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	if template.TenantID == "" {
		template.TenantID = tenantID
	}

	query := `
	INSERT INTO lab_template_profiles (signature_hash, name, base_image, setup_script, resource_profile, tenant_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	ON CONFLICT (signature_hash) DO UPDATE SET
		name = EXCLUDED.name,
		resource_profile = EXCLUDED.resource_profile,
		updated_at = NOW();`

	_, err = r.db.ExecContext(ctx, query, template.SignatureHash, template.Name, template.BaseImage, template.SetupScript, profileJSON, template.TenantID)
	if err != nil {
		return fmt.Errorf("failed to upsert lab template profile: %w", err)
	}

	return nil
}

// UpdateProfileAtomic ejecuta una transacción con SELECT ... FOR UPDATE para garantizar consistencia ante actualizaciones concurrentes
func (r *PostgresLabTemplateRepository) UpdateProfileAtomic(ctx context.Context, signatureHash string, sampleMB float64) (*domain.LabTemplate, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx for EWMA atomic update: %w", err)
	}
	defer tx.Rollback()

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
	SELECT signature_hash, name, base_image, setup_script, resource_profile, tenant_id, created_at, updated_at
	FROM lab_template_profiles
	WHERE signature_hash = $1 AND tenant_id = $2
	FOR UPDATE;`

	var row labTemplateProfileRow
	if err := tx.GetContext(ctx, &row, query, signatureHash, tenantID); err != nil {
		return nil, fmt.Errorf("failed to lock profile for update: %w", err)
	}

	var profile domain.ResourceProfile
	if len(row.ResourceProfile) > 0 {
		_ = json.Unmarshal(row.ResourceProfile, &profile)
	}

	// Aplicación del algoritmo EWMA: S_t = alpha * Y_t + (1 - alpha) * S_{t-1}
	currentEWMA := profile.EWMAState.CurrentEWMAMB
	if currentEWMA <= 0 {
		currentEWMA = sampleMB
	} else {
		currentEWMA = (domain.EWMAAlpha * sampleMB) + ((1.0 - domain.EWMAAlpha) * currentEWMA)
	}

	profile.EWMAState.CurrentEWMAMB = currentEWMA
	profile.EWMAState.SampleCount++
	profile.EWMAState.LastUpdatedAt = time.Now()

	// Recálculo del techo Hard Quota: MaxQuota = EWMA * 1.25 (25% margen de seguridad)
	calculatedQuota := int64(math.Ceil(currentEWMA * domain.EWMASafetyMarginFactor))
	if calculatedQuota < domain.DefaultBaseMemoryMB {
		calculatedQuota = domain.DefaultBaseMemoryMB
	}
	if calculatedQuota > domain.HardQuotaMemoryMB {
		calculatedQuota = domain.HardQuotaMemoryMB
	}
	profile.MaxQuotaMB = calculatedQuota

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated profile json: %w", err)
	}

	updateQuery := `
	UPDATE lab_template_profiles
	SET resource_profile = $1, updated_at = NOW()
	WHERE signature_hash = $2 AND tenant_id = $3;`

	if _, err := tx.ExecContext(ctx, updateQuery, profileJSON, signatureHash, tenantID); err != nil {
		return nil, fmt.Errorf("failed to execute EWMA atomic update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit EWMA update tx: %w", err)
	}

	return &domain.LabTemplate{
		ID:              row.SignatureHash,
		Name:            row.Name,
		BaseImage:       row.BaseImage,
		SetupScript:     row.SetupScript,
		SignatureHash:   row.SignatureHash,
		ResourceProfile: profile,
		TenantID:        row.TenantID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}
