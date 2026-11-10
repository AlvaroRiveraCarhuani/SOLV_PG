package integration

import (
	"testing"

	"solv-backend/internal/infrastructure/database"
)

// approvedJudgeTemplateID devuelve una plantilla de juez aprobada de la base de
// tests. Las plantillas base se siembran con IDs aleatorios en las migraciones,
// así que las pruebas no pueden fijar un UUID: lo consultan.
func approvedJudgeTemplateID(t *testing.T, db *database.Database) string {
	t.Helper()
	var id string
	err := db.GetDB().QueryRow(`
		SELECT id FROM lab_templates
		WHERE status IN ('approved','APROBADA') AND target_environment = 'JUEZ_EFIMERO'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&id)
	if err != nil {
		t.Fatalf("no hay plantilla de juez aprobada en la base de tests: %v", err)
	}
	return id
}
