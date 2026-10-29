package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

type capacityTemplateRepo struct {
	templates map[string]*domain.Template
	errFor    map[string]bool
}

func (r capacityTemplateRepo) GetTemplateByID(ctx context.Context, id string) (*domain.Template, error) {
	if r.errFor[id] {
		return nil, errors.New("template lookup failed")
	}
	return r.templates[id], nil
}

func capacityTemplate(id string, engines ...string) *domain.Template {
	bindings := make([]domain.ServiceRequirement, 0, len(engines))
	for _, engine := range engines {
		bindings = append(bindings, domain.ServiceRequirement{Engine: engine})
	}
	return &domain.Template{
		ID:             id,
		ServicesConfig: domain.ServicesConfig{Services: bindings},
	}
}

func capacityWorkspaces(templateID string, count int) []*domain.WorkspaceInstance {
	workspaces := make([]*domain.WorkspaceInstance, 0, count)
	for i := 0; i < count; i++ {
		id := templateID
		workspaces = append(workspaces, &domain.WorkspaceInstance{TemplateID: &id})
	}
	return workspaces
}

func capacityService(repo domain.TemplateRepository) *WorkspaceService {
	return NewWorkspaceService(nil, nil, nil).WithProvisioning(nil, repo)
}

// La reserva de un satélite es del nodo, no del alumno: es la misma con 2 que con 30
// laboratorios activos, y cada motor se cuenta una sola vez.
func TestNodeSatelliteReserveMB_CountsEachEngineOnce(t *testing.T) {
	repo := capacityTemplateRepo{
		templates: map[string]*domain.Template{
			"tpl-mongo":       capacityTemplate("tpl-mongo", "mongodb"),
			"tpl-mongo-redis": capacityTemplate("tpl-mongo-redis", "mongodb", "redis"),
			"tpl-pg-redis":    capacityTemplate("tpl-pg-redis", "postgres", "redis"),
			"tpl-sqlite":      capacityTemplate("tpl-sqlite", "sqlite"),
		},
		errFor: map[string]bool{"tpl-broken": true},
	}
	svc := capacityService(repo)

	tests := []struct {
		name      string
		requested *domain.Template
		active    []*domain.WorkspaceInstance
		wantMB    int64
	}{
		{name: "sin plantilla", requested: nil, wantMB: 0},
		{name: "sin laboratorios activos", requested: repo.templates["tpl-mongo"], wantMB: 256},
		{name: "dos alumnos comparten el mismo motor", requested: repo.templates["tpl-mongo"], active: capacityWorkspaces("tpl-mongo", 2), wantMB: 256},
		{name: "treinta alumnos comparten el mismo motor", requested: repo.templates["tpl-mongo"], active: capacityWorkspaces("tpl-mongo", 30), wantMB: 256},
		{name: "dos motores declarados", requested: repo.templates["tpl-mongo-redis"], wantMB: 352},
		// PostgreSQL de aula es un satélite dedicado (ADR-051): 384 MB, una vez por nodo.
		{name: "postgres de aula y redis", requested: repo.templates["tpl-pg-redis"], wantMB: 480},
		{name: "dos alumnos comparten el postgres de aula", requested: repo.templates["tpl-pg-redis"], active: capacityWorkspaces("tpl-pg-redis", 2), wantMB: 480},
		{name: "treinta alumnos comparten el postgres de aula", requested: repo.templates["tpl-pg-redis"], active: capacityWorkspaces("tpl-pg-redis", 30), wantMB: 480},
		{name: "sqlite no reserva memoria de nodo", requested: repo.templates["tpl-sqlite"], wantMB: 0},
		{name: "plantilla activa ilegible reserva el conjunto completo", active: capacityWorkspaces("tpl-broken", 1), wantMB: 1248},
		{name: "plantilla activa inexistente reserva el conjunto completo", active: capacityWorkspaces("tpl-ghost", 1), wantMB: 1248},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.nodeSatelliteReserveMB(context.Background(), tt.active, tt.requested); got != tt.wantMB {
				t.Fatalf("reserva esperada %d MB, obtenida %d MB", tt.wantMB, got)
			}
		})
	}
}

// Un motor con destino externo declarado por el operador no consume memoria del nodo.
func TestNodeSatelliteReserveMB_SkipsExternalHost(t *testing.T) {
	t.Setenv("REDIS_SATELLITE_HOST", "redis.example.internal")
	svc := capacityService(capacityTemplateRepo{
		templates: map[string]*domain.Template{
			"tpl-redis": capacityTemplate("tpl-redis", "redis"),
			"tpl-both":  capacityTemplate("tpl-both", "redis", "mongodb"),
		},
	})

	if got := svc.nodeSatelliteReserveMB(context.Background(), nil, capacityTemplate("tpl-redis", "redis")); got != 0 {
		t.Fatalf("Redis externo no debe reservar memoria del nodo, obtenido %d MB", got)
	}
	if got := svc.nodeSatelliteReserveMB(context.Background(), nil, capacityTemplate("tpl-both", "redis", "mongodb")); got != 256 {
		t.Fatalf("solo MongoDB debe reservar, obtenido %d MB", got)
	}
}

func TestAdmissionBudgetDenied(t *testing.T) {
	tests := []struct {
		name        string
		committedMB int64
		labLimitMB  int64
		reserveMB   int64
		hostMB      int64
		ceilingPct  int64
		want        bool
	}{
		{
			// 2 alumnos × 512 MB + 256 MB de MongoDB sobre un nodo de 8 GB: entra.
			name: "dos alumnos con mongo entran", committedMB: 1024, labLimitMB: 512, reserveMB: 256,
			hostMB: 8192, ceilingPct: 85, want: false,
		},
		{
			// El alumno 30 sobre un nodo de 18 GB: sin satélites entra, con MongoDB y
			// Redis reservados una sola vez ya no.
			name: "treinta alumnos sin satélites", committedMB: 29 * 512, labLimitMB: 512, reserveMB: 0,
			hostMB: 18432, ceilingPct: 85, want: false,
		},
		{
			name: "treinta alumnos con mongo y redis", committedMB: 29 * 512, labLimitMB: 512, reserveMB: 352,
			hostMB: 18432, ceilingPct: 85, want: true,
		},
		{
			// La cuenta anterior (768 MB por laboratorio por el delta del satélite)
			// negaba el décimo alumno en un nodo de 8 GB: 9 × 768 + 768 > 6963.
			name: "la cuenta vieja negaba el décimo alumno", committedMB: 9 * 768, labLimitMB: 768, reserveMB: 0,
			hostMB: 8192, ceilingPct: 85, want: true,
		},
		{
			// Con la cuota del laboratorio sin el delta, el mismo décimo alumno entra.
			name: "la cuenta nueva admite el décimo alumno", committedMB: 9 * 512, labLimitMB: 512, reserveMB: 256,
			hostMB: 8192, ceilingPct: 85, want: false,
		},
		{
			name: "sin host no se decide", committedMB: 100, labLimitMB: 100, reserveMB: 100,
			hostMB: 0, ceilingPct: 85, want: false,
		},
		{
			// Sin techo configurado rige el 85%: 8192 × 85% = 6963.
			name: "techo por defecto", committedMB: 7000, labLimitMB: 512, reserveMB: 0,
			hostMB: 8192, ceilingPct: 0, want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := admissionBudgetDenied(tt.committedMB, tt.labLimitMB, tt.reserveMB, tt.hostMB, tt.ceilingPct)
			if got != tt.want {
				t.Fatalf("denegado esperado %v, obtenido %v", tt.want, got)
			}
		})
	}
}
