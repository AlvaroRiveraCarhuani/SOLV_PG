package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

type mockExerciseRepo struct {
	exercise *domain.Exercise
}

func (m *mockExerciseRepo) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return m.exercise, nil
}

func (m *mockExerciseRepo) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return m.exercise, nil
}

func (m *mockExerciseRepo) Create(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}

func (m *mockExerciseRepo) Update(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}

func (m *mockExerciseRepo) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	return nil
}

func (m *mockExerciseRepo) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	return nil
}

func (m *mockExerciseRepo) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}

func (m *mockExerciseRepo) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return []*domain.DueAssignment{}, nil
}

func (m *mockExerciseRepo) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	return []*domain.Exercise{}, nil
}

type mockRunner struct{}

func (m *mockRunner) RunTestCase(ctx context.Context, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	return domain.TestCaseRunResult{
		Verdict:       domain.VerdictAC,
		ExecutionTime: 5 * time.Millisecond,
		ActualOutput:  "Hello World\n",
	}, nil
}

func (m *mockRunner) RunDBDryRun(ctx context.Context, config domain.DBEvaluationRunConfig) (string, error) {
	return "{}", nil
}

func (m *mockRunner) RunDBEvaluation(ctx context.Context, config domain.DBEvaluationRunConfig) (domain.DBEvaluationResult, error) {
	return domain.DBEvaluationResult{Verdict: domain.VerdictAC}, nil
}

func getRulesDir() string {
	// Try relative from test execution directory
	path := filepath.Join("..", "..", "internal", "infrastructure", "semgrep", "rules")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return filepath.Join("internal", "infrastructure", "semgrep", "rules")
}

func TestSemgrepPrecheckSuite(t *testing.T) {
	if _, err := exec.LookPath("semgrep"); err != nil {
		t.Skip("Skipping Semgrep CLI tests: semgrep executable not found in PATH")
	}
	rulesDir := getRulesDir()
	worker := services.NewSemgrepWorker(nil, nil, rulesDir)
	astAnalyzer := services.NewStaticASTAnalyzer()

	exercise := &domain.Exercise{
		ID:   "ex-test-1",
		Type: domain.ExerciseTypeAlgorithm,
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TestCases: domain.TestCases{
					{Input: "", ExpectedOutput: "Hello World\n", IsHidden: false},
				},
				TimeLimitMS:   1000,
				MemoryLimitMB: 128,
			},
		},
	}

	evalService := services.NewEvaluationService(&mockExerciseRepo{exercise: exercise}, astAnalyzer, worker, &mockRunner{})
	ctx := context.Background()

	t.Run("1. Python con import prohibido -> AST_BLOCKED", func(t *testing.T) {
		code := "import os\nprint('Hello')"
		b64Code := base64.StdEncoding.EncodeToString([]byte(code))

		start := time.Now()
		res, err := evalService.Evaluate(ctx, exercise.ID, "python", b64Code)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Verdict != domain.VerdictASTBlocked {
			t.Errorf("Expected verdict AST_BLOCKED, got %s (message: %s)", res.Verdict, res.Message)
		}
		if latency > 100*time.Millisecond {
			t.Logf("WARNING: Pre-check latency was %v (>100ms SLA)", latency)
		} else {
			t.Logf("PASS: Pre-check latency: %v (<100ms SLA met)", latency)
		}
	})

	t.Run("2. JavaScript con eval -> AST_BLOCKED", func(t *testing.T) {
		code := "const result = eval('1 + 1');\nconsole.log(result);"
		b64Code := base64.StdEncoding.EncodeToString([]byte(code))

		start := time.Now()
		res, err := evalService.Evaluate(ctx, exercise.ID, "javascript", b64Code)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Verdict != domain.VerdictASTBlocked {
			t.Errorf("Expected verdict AST_BLOCKED, got %s (message: %s)", res.Verdict, res.Message)
		}
		t.Logf("PASS: JS eval pre-check latency: %v", latency)
	})

	t.Run("3. Java con Runtime.exec -> AST_BLOCKED", func(t *testing.T) {
		code := "public class Solution {\n public static void main(String[] args) throws Exception {\n Runtime.getRuntime().exec(\"ls\");\n }\n}"
		b64Code := base64.StdEncoding.EncodeToString([]byte(code))

		start := time.Now()
		res, err := evalService.Evaluate(ctx, exercise.ID, "java", b64Code)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Verdict != domain.VerdictASTBlocked {
			t.Errorf("Expected verdict AST_BLOCKED, got %s (message: %s)", res.Verdict, res.Message)
		}
		t.Logf("PASS: Java Runtime.exec pre-check latency: %v", latency)
	})

	t.Run("4. Código limpio -> Pasa pre-chequeo y obtiene AC", func(t *testing.T) {
		code := "print('Hello World')"
		b64Code := base64.StdEncoding.EncodeToString([]byte(code))

		start := time.Now()
		res, err := evalService.Evaluate(ctx, exercise.ID, "python", b64Code)
		latency := time.Since(start)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.Verdict != domain.VerdictAC {
			t.Errorf("Expected verdict AC, got %s (message: %s)", res.Verdict, res.Message)
		}
		t.Logf("PASS: Clean code passed pre-check and received AC in %v", latency)
	})

	t.Run("5. Persistencia de ast_result en PostgreSQL", func(t *testing.T) {
		dbDSN := os.Getenv("DATABASE_URL")
		if dbDSN == "" {
			dbDSN = "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
		}

		db, err := database.NewPostgresDB(dbDSN)
		if err != nil {
			t.Skipf("Skipping DB persistence test: unable to connect: %v", err)
		}

		sqlDB := db.GetDB()
		subRepo := postgres.NewPostgresSubmissionRepository(sqlDB)

		tenantID := "90000000-0000-0000-0000-000000000001"
		exerciseID := "90000000-0000-0000-0000-000000000002"
		studentID := "90000000-0000-0000-0000-000000000003"
		subID := "90000000-0000-0000-0000-000000000004"

		// Clean up previous test entries
		sqlDB.Exec("DELETE FROM submissions WHERE id = $1", subID)
		sqlDB.Exec("DELETE FROM exercises WHERE id = $1", exerciseID)
		sqlDB.Exec("DELETE FROM users WHERE id = $1", studentID)
		sqlDB.Exec("DELETE FROM tenants WHERE id = $1", tenantID)

		// Insert required FK records
		if _, err := sqlDB.Exec("INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", tenantID, "Semgrep Test Tenant", "semgrep-test-tenant"); err != nil {
			t.Fatalf("Failed to insert tenant: %v", err)
		}
		if _, err := sqlDB.Exec("INSERT INTO users (id, tenant_id, email, first_name, last_name, role) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING", studentID, tenantID, "semgrep-student@test.com", "Semgrep", "Student", "student"); err != nil {
			t.Fatalf("Failed to insert user: %v", err)
		}
		if _, err := sqlDB.Exec("INSERT INTO exercises (id, tenant_id, title, type, config) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO NOTHING", exerciseID, tenantID, "Semgrep Ex", "algorithm", "{}"); err != nil {
			t.Fatalf("Failed to insert exercise: %v", err)
		}

		scanResult := &domain.ScanResult{
			HasViolations: true,
			Violations: []domain.ScanViolation{
				{
					RuleID:  "solv-python-forbidden-import-os",
					Message: "Import of 'os' module is forbidden for security reasons",
					Line:    1,
				},
			},
		}
		rawJSON, err := json.Marshal(scanResult)
		if err != nil {
			t.Fatalf("Failed to marshal scanResult: %v", err)
		}

		sub := &domain.Submission{
			ID:              subID,
			TenantID:        tenantID,
			ExerciseID:      exerciseID,
			StudentID:       studentID,
			Code:            "import os",
			Verdict:         string(domain.VerdictASTBlocked),
			ASTResult:       rawJSON,
			ExecutionTimeMS: 0,
			MemoryUsedMB:    0,
			SubmittedAt:     time.Now(),
		}

		if err := subRepo.Create(ctx, sub); err != nil {
			t.Fatalf("Failed to create submission with ASTResult: %v", err)
		}

		fetched, err := subRepo.GetByID(ctx, sub.TenantID, sub.ID)
		if err != nil {
			t.Fatalf("Failed to fetch created submission: %v", err)
		}

		if fetched.Verdict != string(domain.VerdictASTBlocked) {
			t.Errorf("Expected verdict %s, got %s", domain.VerdictASTBlocked, fetched.Verdict)
		}

		var fetchedScanRes domain.ScanResult
		if err := json.Unmarshal(fetched.ASTResult, &fetchedScanRes); err != nil {
			t.Fatalf("Failed to unmarshal stored ast_result JSONB: %v", err)
		}

		if !fetchedScanRes.HasViolations || len(fetchedScanRes.Violations) == 0 {
			t.Errorf("Expected violations in stored ast_result, got none")
		} else {
			t.Logf("PASS: ast_result JSONB persisted and retrieved successfully! Stored RuleID: %s", fetchedScanRes.Violations[0].RuleID)
		}

		// Clean up
		sqlDB.Exec("DELETE FROM submissions WHERE id = $1", subID)
		sqlDB.Exec("DELETE FROM exercises WHERE id = $1", exerciseID)
		sqlDB.Exec("DELETE FROM users WHERE id = $1", studentID)
		sqlDB.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	})
}
