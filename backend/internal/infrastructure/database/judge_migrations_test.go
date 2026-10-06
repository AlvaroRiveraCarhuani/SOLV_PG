package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Base directory of goose migrations relative to this package.
func judgeMigrationsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "migrations")
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("migrations dir not reachable: %v", err)
	}
	return dir
}

func readJudgeMigration(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("migration %s missing: %v", name, err)
	}
	content := string(raw)
	if !strings.Contains(content, "+goose Up") {
		t.Errorf("migration %s lacks +goose Up block", name)
	}
	if !strings.Contains(content, "+goose Down") {
		t.Errorf("migration %s lacks +goose Down block", name)
	}
	return content
}

func TestJudgeMigrationsExist(t *testing.T) {
	dir := judgeMigrationsDir(t)
	files := []string{
		"00009_judge_comparator_spec.sql",
		"00010_judge_reference_stale.sql",
		"00011_judge_language_profiles.sql",
		"00012_judge_run_metrics.sql",
		"00013_exercise_metadata_and_test_cases_schema.sql",
		"00014_migrate_jsonb_test_cases_to_table.sql",
		"00015_cleanup_jsonb_test_cases.sql",
		"00016_course_modules_and_prerequisites.sql",
	}
	for _, name := range files {
		content := readJudgeMigration(t, dir, name)
		if strings.TrimSpace(content) == "" {
			t.Errorf("migration %s is empty", name)
		}
	}
}

func TestJudgeMigration0016CourseModulesAndPrerequisites(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00016_course_modules_and_prerequisites.sql")
	for _, marker := range []string{
		"course_modules",
		"module_prerequisites",
		"module_id",
		"idx_exercises_module",
		"+goose Down",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("migration 00016 lacks expected marker %q", marker)
		}
	}
}

func TestJudgeMigration0013MetadataAndSchema(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00013_exercise_metadata_and_test_cases_schema.sql")
	for _, marker := range []string{
		"difficulty",
		"tags",
		"purpose",
		"per_student_seed",
		"exercise_test_cases",
		"visibility",
		"weight",
		"UNIQUE (exercise_id, order_index)",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("00013 missing marker %q", marker)
		}
	}
}

func TestJudgeMigration0014DataMigration(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00014_migrate_jsonb_test_cases_to_table.sql")
	for _, marker := range []string{
		"INSERT INTO exercise_test_cases",
		"jsonb_array_elements",
		"WITH ORDINALITY",
		"ON CONFLICT (exercise_id, order_index) DO UPDATE",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("00014 missing marker %q", marker)
		}
	}
}

func TestJudgeMigration0015JSONBCleanup(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00015_cleanup_jsonb_test_cases.sql")
	for _, marker := range []string{
		"UPDATE exercises",
		"config #- '{algorithm,test_cases}'",
		"jsonb_agg",
		"jsonb_build_object",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("00015 missing marker %q", marker)
		}
	}
}

func TestJudgeMigration0009ComparatorSpec(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00009_judge_comparator_spec.sql")
	for _, marker := range []string{
		"comparator_spec",
		`{"id":"exact"}`,
		"NOT NULL",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("00009 missing marker %q", marker)
		}
	}
}

func TestJudgeMigration0010ReferenceStale(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00010_judge_reference_stale.sql")
	for _, marker := range []string{
		"reference_solution",
		"stale",
		"last_valid_dry_run_at",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("00010 missing marker %q", marker)
		}
	}
}

func TestJudgeMigration0011ProfilesSeed(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00011_judge_language_profiles.sql")
	for _, marker := range []string{
		"language_profiles",
		"p95_window_days",
		"DEFAULT 30",
		"checker_sidecar_image",
		"language_profile_audits",
		"@sha256:",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("00011 missing marker %q", marker)
		}
	}
	seeds := []struct {
		language string
		timeout  string
		memory   string
	}{
		{"python", "2000", "256"},
		{"javascript", "2000", "256"},
		{"cpp", "1000", "128"},
		{"c", "1000", "128"},
		{"csharp", "2500", "256"},
		{"java", "3000", "512"},
	}
	for _, seed := range seeds {
		if !strings.Contains(content, "'"+seed.language+"'") {
			t.Errorf("00011 seed missing language %q", seed.language)
		}
		row := seed.language + "', " + seed.timeout + ", " + seed.memory
		if !strings.Contains(content, row) {
			t.Errorf("00011 seed wrong limits for %s: want %q", seed.language, row)
		}
	}
}

func TestJudgeMigration0012MetricsJobs(t *testing.T) {
	dir := judgeMigrationsDir(t)
	content := readJudgeMigration(t, dir, "00012_judge_run_metrics.sql")
	for _, marker := range []string{
		"run_metrics",
		"image_digest",
		"duration_ms",
		"verdict",
		"dry_run_jobs",
		"queued",
		"running",
		"done",
		"failed",
	} {
		if !strings.Contains(content, marker) {
			t.Errorf("00012 missing marker %q", marker)
		}
	}
}
