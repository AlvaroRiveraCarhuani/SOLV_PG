package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

// Una acción de emergencia sin executor no se ejecutó: tiene que informarse como
// falta de executor y no como éxito con cero afectados.
func TestExecuteEmergencyAction_SinExecutorInformaFalta(t *testing.T) {
	service := NewAdminGovernanceService(nil, nil)
	ctx := context.Background()

	for _, action := range []string{ActionKillZombies, ActionDockerPrune, ActionResetPools} {
		t.Run(action, func(t *testing.T) {
			phrase, ok := emergencyConfirmationPhrase(action)
			if !ok {
				t.Fatalf("la acción %s no tiene frase de confirmación", action)
			}

			result, err := service.ExecuteEmergencyAction(ctx, "tenant-1", "admin-1", action,
				domain.EmergencyActionRequest{ConfirmationPhrase: phrase})

			if !errors.Is(err, ErrEmergencyExecutorUnavailable) {
				t.Fatalf("se esperaba ErrEmergencyExecutorUnavailable, se obtuvo %v", err)
			}
			if result != nil {
				t.Errorf("no debe devolverse un resultado cuando la acción no se ejecutó: %+v", result)
			}
		})
	}
}

func TestExecuteEmergencyAction_RechazaFraseIncorrecta(t *testing.T) {
	service := NewAdminGovernanceService(nil, nil)

	_, err := service.ExecuteEmergencyAction(context.Background(), "tenant-1", "admin-1",
		ActionKillZombies, domain.EmergencyActionRequest{ConfirmationPhrase: "limpiar zombies"})

	if !errors.Is(err, ErrInvalidConfirmationPhrase) {
		t.Fatalf("se esperaba ErrInvalidConfirmationPhrase, se obtuvo %v", err)
	}
}

func TestExecuteEmergencyAction_RechazaAccionDesconocida(t *testing.T) {
	service := NewAdminGovernanceService(nil, nil)

	_, err := service.ExecuteEmergencyAction(context.Background(), "tenant-1", "admin-1",
		"borrar_todo", domain.EmergencyActionRequest{ConfirmationPhrase: "BORRAR TODO"})

	if !errors.Is(err, ErrUnknownEmergencyAction) {
		t.Fatalf("se esperaba ErrUnknownEmergencyAction, se obtuvo %v", err)
	}
}

func TestExecuteEmergencyAction_InformaLoQueBarrioElExecutor(t *testing.T) {
	service := NewAdminGovernanceService(nil, nil)
	service.SetDockerPruner(func(ctx context.Context) (int64, int64, error) {
		return 7, 0, nil
	})

	result, err := service.ExecuteEmergencyAction(context.Background(), "tenant-1", "admin-1",
		ActionKillZombies, domain.EmergencyActionRequest{ConfirmationPhrase: PhraseKillZombies})

	if err != nil {
		t.Fatalf("la acción debía ejecutarse: %v", err)
	}
	if result == nil || result.AffectedCount != 7 {
		t.Fatalf("el resultado debe reflejar al executor: %+v", result)
	}
}

func TestExecuteEmergencyAction_ErrorDelExecutorNoSeReportaComoExito(t *testing.T) {
	service := NewAdminGovernanceService(nil, nil)
	falla := errors.New("daemon no responde")
	service.SetDockerPruner(func(ctx context.Context) (int64, int64, error) {
		return 0, 0, falla
	})

	result, err := service.ExecuteEmergencyAction(context.Background(), "tenant-1", "admin-1",
		ActionKillZombies, domain.EmergencyActionRequest{ConfirmationPhrase: PhraseKillZombies})

	if !errors.Is(err, falla) {
		t.Fatalf("el error del executor debe propagarse, se obtuvo %v", err)
	}
	if result != nil {
		t.Errorf("no debe devolverse resultado si el executor falló: %+v", result)
	}
}
