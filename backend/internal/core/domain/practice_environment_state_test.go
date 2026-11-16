package domain

import (
	"encoding/json"
	"testing"
	"time"
)

// El estado visible sale del registro: cerrado manda sobre todo, después el
// fallo, después lo que todavía no está listo.
func TestPracticeEnvironmentState(t *testing.T) {
	frozenAt := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		nombre   string
		env      PracticeEnvironment
		esperado PracticeEnvironmentState
	}{
		{
			nombre:   "pendiente está preparando",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusPending},
			esperado: PracticeEnvironmentStatePreparing,
		},
		{
			nombre:   "corriendo sin proyecto está listo",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, WorkspaceSeedMode: WorkspaceSeedModeNone},
			esperado: PracticeEnvironmentStateReady,
		},
		{
			nombre:   "corriendo con proyecto sembrado está listo",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, WorkspaceSeedMode: WorkspaceSeedModeProject, WorkspaceInitStatus: WorkspaceInitStatusSetupDone},
			esperado: PracticeEnvironmentStateReady,
		},
		{
			nombre:   "corriendo con siembra pendiente sigue preparando",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, WorkspaceSeedMode: WorkspaceSeedModeProject, WorkspaceInitStatus: WorkspaceInitStatusPending},
			esperado: PracticeEnvironmentStatePreparing,
		},
		{
			nombre:   "la cuarentena de la siembra es un fallo",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, WorkspaceSeedMode: WorkspaceSeedModeProject, WorkspaceInitStatus: WorkspaceInitStatusQuarantined},
			esperado: PracticeEnvironmentStateFailed,
		},
		{
			nombre:   "un volumen perdido es un fallo",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, WorkspaceSeedMode: WorkspaceSeedModeProject, WorkspaceInitStatus: WorkspaceInitStatusMissing},
			esperado: PracticeEnvironmentStateFailed,
		},
		{
			nombre:   "el aviso de disco no cierra el editor",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, DiskState: "warning", DiskUsedMB: 840, DiskLimitMB: 1024},
			esperado: PracticeEnvironmentStateDiskWarning,
		},
		{
			nombre:   "el disco lleno frena la entrega sin cerrar",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, DiskState: "full", DiskUsedMB: 1024, DiskLimitMB: 1024},
			esperado: PracticeEnvironmentStateDiskFull,
		},
		{
			nombre:   "congelado está cerrado",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusRunning, FrozenAt: &frozenAt},
			esperado: PracticeEnvironmentStateClosed,
		},
		{
			nombre:   "el cierre manda sobre el fallo",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusFailed, FrozenAt: &frozenAt},
			esperado: PracticeEnvironmentStateClosed,
		},
		{
			nombre:   "fallido",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusFailed},
			esperado: PracticeEnvironmentStateFailed,
		},
		{
			nombre:   "muerto por memoria",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusOOMKilled},
			esperado: PracticeEnvironmentStateFailed,
		},
		{
			nombre:   "hibernado se está despertando",
			env:      PracticeEnvironment{Status: PracticeEnvironmentStatusHibernated},
			esperado: PracticeEnvironmentStatePreparing,
		},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			if obtenido := tc.env.State(); obtenido != tc.esperado {
				t.Fatalf("esperado %q, obtenido %q", tc.esperado, obtenido)
			}
		})
	}
}

// La respuesta del entorno lleva el estado calculado sin tocar los campos que se
// guardan.
func TestPracticeEnvironmentJSONIncluyeElEstado(t *testing.T) {
	env := PracticeEnvironment{ID: "env-1", Status: PracticeEnvironmentStatusRunning, WorkspaceSeedMode: WorkspaceSeedModeNone}

	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("no se pudo serializar: %v", err)
	}
	var decoded struct {
		ID     string `json:"environment_id"`
		Status string `json:"status"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}
	if decoded.ID != "env-1" || decoded.Status != PracticeEnvironmentStatusRunning || decoded.State != string(PracticeEnvironmentStateReady) {
		t.Fatalf("respuesta inesperada: %s", raw)
	}
}
