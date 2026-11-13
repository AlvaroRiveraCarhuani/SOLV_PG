package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExerciseConfigValue_NoPersisteLosScriptsDeInicializacion(t *testing.T) {
	const script = "CREATE TABLE materias (id INT PRIMARY KEY);"
	config := ExerciseConfig{
		IDE: &IDEConfig{
			AllowResubmit:  true,
			AutoCheckpoint: true,
			DBInitScript:   script,
		},
		Database: &DatabaseConfig{Engine: "postgres", InitScript: script},
	}

	value, err := config.Value()
	if err != nil {
		t.Fatalf("Value() devolvió error: %v", err)
	}
	raw, ok := value.([]byte)
	if !ok {
		t.Fatalf("Value() debe devolver []byte, obtuvo %T", value)
	}

	persisted := string(raw)
	if strings.Contains(persisted, "db_init_script") {
		t.Errorf("el config persistido no puede contener el script del docente: %s", persisted)
	}
	if strings.Contains(persisted, "init_script") {
		t.Errorf("el config persistido no puede contener el espejo del script: %s", persisted)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("el config persistido debe ser JSON válido: %v", err)
	}
	ide, ok := decoded["ide"].(map[string]any)
	if !ok {
		t.Fatalf("el config persistido debe conservar el bloque ide: %s", persisted)
	}
	if ide["allow_resubmit"] != true || ide["auto_checkpoint"] != true {
		t.Errorf("el config persistido debe conservar la política de entrega: %s", persisted)
	}
}

func TestExerciseConfigValue_ConservaElScriptEnMemoria(t *testing.T) {
	const script = "CREATE TABLE t (id INT);"
	config := ExerciseConfig{
		IDE:      &IDEConfig{DBInitScript: script},
		Database: &DatabaseConfig{InitScript: script},
	}

	if _, err := config.Value(); err != nil {
		t.Fatalf("Value() devolvió error: %v", err)
	}

	if config.IDE.DBInitScript != script {
		t.Error("Value() no puede vaciar el script en memoria: el arranque del laboratorio lo necesita")
	}
	if config.Database.InitScript != script {
		t.Error("Value() no puede vaciar el espejo en memoria")
	}
}
