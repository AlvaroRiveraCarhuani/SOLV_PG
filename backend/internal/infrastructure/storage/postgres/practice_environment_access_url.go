package postgres

import (
	"context"
	"fmt"
)

// UpdateAccessURL persiste el access_url resuelto al recrear el contenedor del
// environment. No forma parte de domain.PracticeEnvironmentRepository para no ampliar la
// interfaz (y no romper sus mocks): el servicio la resuelve por type-assertion.
func (r *PostgresPracticeEnvironmentRepository) UpdateAccessURL(ctx context.Context, id, accessURL string) error {
	query := `UPDATE environments SET access_url = $1, updated_at = NOW() WHERE id = $2`
	if _, err := r.db.ExecContext(ctx, query, accessURL, id); err != nil {
		return fmt.Errorf("failed to update access_url for environment %s: %w", id, err)
	}
	return nil
}
