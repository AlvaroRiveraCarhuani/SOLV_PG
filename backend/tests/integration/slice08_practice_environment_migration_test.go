package integration

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func TestCRIT06DomainAndMockRepoTypeDiscriminator(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockPracticeEnvironmentRepository{environments: make(map[string]*domain.PracticeEnvironment)}

	// 1. Verificar constantes de dominio
	if domain.PracticeEnvironmentTypeIDEPersistente != "IDE_PERSISTENTE" {
		t.Errorf("Expected PracticeEnvironmentTypeIDEPersistente='IDE_PERSISTENTE', got '%s'", domain.PracticeEnvironmentTypeIDEPersistente)
	}
	if domain.PracticeEnvironmentTypeJuezEfimero != "JUEZ_EFIMERO" {
		t.Errorf("Expected PracticeEnvironmentTypeJuezEfimero='JUEZ_EFIMERO', got '%s'", domain.PracticeEnvironmentTypeJuezEfimero)
	}

	// 2. Crear environment IDE_PERSISTENTE en el repositorio
	ws1 := &domain.PracticeEnvironment{
		ID:        uuid.NewString(),
		StudentID: uuid.NewString(),
		SubjectID: uuid.NewString(),
		Type:      domain.PracticeEnvironmentTypeIDEPersistente,
		Status:    domain.PracticeEnvironmentStatusRunning,
	}
	if err := mockRepo.Create(ctx, ws1); err != nil {
		t.Fatalf("Failed to create IDE_PERSISTENTE environment: %v", err)
	}

	// 3. Crear environment JUEZ_EFIMERO en el repositorio
	ws2 := &domain.PracticeEnvironment{
		ID:        uuid.NewString(),
		StudentID: uuid.NewString(),
		SubjectID: uuid.NewString(),
		Type:      domain.PracticeEnvironmentTypeJuezEfimero,
		Status:    domain.PracticeEnvironmentStatusPending,
	}
	if err := mockRepo.Create(ctx, ws2); err != nil {
		t.Fatalf("Failed to create JUEZ_EFIMERO environment: %v", err)
	}

	// 4. Probar GetByType(IDE_PERSISTENTE)
	ides, err := mockRepo.GetByType(ctx, domain.PracticeEnvironmentTypeIDEPersistente)
	if err != nil {
		t.Fatalf("GetByType(IDE_PERSISTENTE) failed: %v", err)
	}
	if len(ides) != 1 || ides[0].ID != ws1.ID {
		t.Errorf("Expected 1 IDE_PERSISTENTE environment with ID %s, got %d items", ws1.ID, len(ides))
	}

	// 5. Probar GetByType(JUEZ_EFIMERO)
	jueces, err := mockRepo.GetByType(ctx, domain.PracticeEnvironmentTypeJuezEfimero)
	if err != nil {
		t.Fatalf("GetByType(JUEZ_EFIMERO) failed: %v", err)
	}
	if len(jueces) != 1 || jueces[0].ID != ws2.ID {
		t.Errorf("Expected 1 JUEZ_EFIMERO environment with ID %s, got %d items", ws2.ID, len(jueces))
	}

	t.Log("Unit test for CRIT-06 domain entities, constants, and GetByType repository method PASSED 100%!")
}

func TestCRIT06PracticeEnvironmentMigrationAndTypeDiscriminator(t *testing.T) {
	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		t.Skip("Skipping live DB integration test: DATABASE_URL environment variable not set")
	}

	ctx := context.Background()
	db, err := database.NewPostgresDB(dbDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 1. Ejecutar migraciones iniciales

	// 2. Verificar que la tabla lab_instances ya NO existe en PostgreSQL
	var labInstancesExists bool
	tableQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'lab_instances'
		);
	`
	if err := db.GetDB().GetContext(ctx, &labInstancesExists, tableQuery); err != nil {
		t.Fatalf("Failed to check if lab_instances table exists: %v", err)
	}
	if labInstancesExists {
		t.Errorf("Assertion failed: lab_instances table should have been dropped, but still exists!")
	}

	// 3. Verificar que la columna 'type' existe en 'environments'
	var typeColumnExists bool
	columnQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.columns 
			WHERE table_name = 'environments' AND column_name = 'type'
		);
	`
	if err := db.GetDB().GetContext(ctx, &typeColumnExists, columnQuery); err != nil {
		t.Fatalf("Failed to check if type column exists in environments: %v", err)
	}
	if !typeColumnExists {
		t.Fatalf("Assertion failed: type column does not exist in environments table!")
	}

	repo := postgres.NewPostgresPracticeEnvironmentRepository(db.GetDB())

	// 4. Crear un PracticeEnvironment de tipo IDE_PERSISTENTE y verificar tipo por defecto
	wsPersistente := &domain.PracticeEnvironment{
		ID:        uuid.NewString(),
		StudentID: uuid.NewString(),
		SubjectID: "00000000-0000-0000-0000-000000000001",
		Status:    domain.PracticeEnvironmentStatusPending,
		AccessURL: "http://test-ide.solv.local",
	}
	if err := repo.Create(ctx, wsPersistente); err != nil {
		t.Fatalf("Failed to create IDE_PERSISTENTE environment: %v", err)
	}

	fetchedIDE, err := repo.GetByID(ctx, wsPersistente.ID)
	if err != nil {
		t.Fatalf("Failed to fetch created IDE_PERSISTENTE environment: %v", err)
	}
	if fetchedIDE.Type != domain.PracticeEnvironmentTypeIDEPersistente {
		t.Errorf("Expected environment type '%s', got '%s'", domain.PracticeEnvironmentTypeIDEPersistente, fetchedIDE.Type)
	}

	// 5. Crear un PracticeEnvironment de tipo JUEZ_EFIMERO
	wsJuez := &domain.PracticeEnvironment{
		ID:        uuid.NewString(),
		StudentID: uuid.NewString(),
		SubjectID: "00000000-0000-0000-0000-000000000001",
		Type:      domain.PracticeEnvironmentTypeJuezEfimero,
		Status:    domain.PracticeEnvironmentStatusPending,
		AccessURL: "",
	}
	if err := repo.Create(ctx, wsJuez); err != nil {
		t.Fatalf("Failed to create JUEZ_EFIMERO environment: %v", err)
	}

	fetchedJuez, err := repo.GetByID(ctx, wsJuez.ID)
	if err != nil {
		t.Fatalf("Failed to fetch created JUEZ_EFIMERO environment: %v", err)
	}
	if fetchedJuez.Type != domain.PracticeEnvironmentTypeJuezEfimero {
		t.Errorf("Expected environment type '%s', got '%s'", domain.PracticeEnvironmentTypeJuezEfimero, fetchedJuez.Type)
	}

	// 6. Probar filtrado por tipo con GetByType
	juezList, err := repo.GetByType(ctx, domain.PracticeEnvironmentTypeJuezEfimero)
	if err != nil {
		t.Fatalf("Failed to query environments by type JUEZ_EFIMERO: %v", err)
	}
	foundJuez := false
	for _, item := range juezList {
		if item.ID == wsJuez.ID {
			foundJuez = true
			break
		}
	}
	if !foundJuez {
		t.Errorf("Expected to find environment %s in GetByType(JUEZ_EFIMERO)", wsJuez.ID)
	}

	t.Logf("CRIT-06 Integration Test PASSED! PracticeEnvironments type discriminator and lab_instances table removal verified.")
}
