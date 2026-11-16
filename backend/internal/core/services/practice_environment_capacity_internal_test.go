package services

import (
	"context"
	"errors"
	"fmt"
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

func capacityPracticeEnvironments(templateID string, count int) []*domain.PracticeEnvironment {
	environments := make([]*domain.PracticeEnvironment, 0, count)
	for i := 0; i < count; i++ {
		id := templateID
		environments = append(environments, &domain.PracticeEnvironment{TemplateID: &id})
	}
	return environments
}

func capacityService(repo domain.TemplateRepository) *PracticeEnvironmentService {
	return NewPracticeEnvironmentService(nil, nil, nil).WithProvisioning(nil, repo)
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
		active    []*domain.PracticeEnvironment
		wantMB    int64
	}{
		{name: "sin plantilla", requested: nil, wantMB: 0},
		{name: "sin laboratorios activos", requested: repo.templates["tpl-mongo"], wantMB: 256},
		{name: "dos alumnos comparten el mismo motor", requested: repo.templates["tpl-mongo"], active: capacityPracticeEnvironments("tpl-mongo", 2), wantMB: 256},
		{name: "treinta alumnos comparten el mismo motor", requested: repo.templates["tpl-mongo"], active: capacityPracticeEnvironments("tpl-mongo", 30), wantMB: 256},
		{name: "dos motores declarados", requested: repo.templates["tpl-mongo-redis"], wantMB: 352},
		// PostgreSQL de aula es un satélite dedicado (ADR-051): 384 MB, una vez por nodo.
		{name: "postgres de aula y redis", requested: repo.templates["tpl-pg-redis"], wantMB: 480},
		{name: "dos alumnos comparten el postgres de aula", requested: repo.templates["tpl-pg-redis"], active: capacityPracticeEnvironments("tpl-pg-redis", 2), wantMB: 480},
		{name: "treinta alumnos comparten el postgres de aula", requested: repo.templates["tpl-pg-redis"], active: capacityPracticeEnvironments("tpl-pg-redis", 30), wantMB: 480},
		{name: "sqlite no reserva memoria de nodo", requested: repo.templates["tpl-sqlite"], wantMB: 0},
		{name: "plantilla activa ilegible reserva el conjunto completo", active: capacityPracticeEnvironments("tpl-broken", 1), wantMB: 1248},
		{name: "plantilla activa inexistente reserva el conjunto completo", active: capacityPracticeEnvironments("tpl-ghost", 1), wantMB: 1248},
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

// La cuota de un entorno es su base más el préstamo que permita el hueco del
// nodo, y el préstamo siempre deja sitio para otro entorno del mismo tamaño.
// Devuelve 0 cuando ni la base entra: la admisión rechaza el entorno.
func TestLoanQuotaMB(t *testing.T) {
	tests := []struct {
		name        string
		baseMB      int64
		committedMB int64
		reserveMB   int64
		hostMB      int64
		ceilingPct  int64
		hardQuotaMB int64
		want        int64
	}{
		{
			// Nodo de 12 GB vacío: 10200 MB de techo, la base se dobla hasta 4096.
			name: "nodo vacío presta hasta el techo duro", baseMB: 512,
			hostMB: 12000, ceilingPct: 85, hardQuotaMB: 4096, want: 4096,
		},
		{
			// Con otro entorno de 4096 abierto quedan 6104 MB: caben dos de 2048.
			name: "con un entorno comprometido presta 2048", baseMB: 512, committedMB: 4096,
			hostMB: 12000, ceilingPct: 85, hardQuotaMB: 4096, want: 2048,
		},
		{
			name: "con dos entornos comprometidos presta 1536", baseMB: 512, committedMB: 6144,
			hostMB: 12000, ceilingPct: 85, hardQuotaMB: 4096, want: 1536,
		},
		{
			// La cuenta de satélites no se multiplica por alumno, pero sí reduce
			// el hueco del nodo.
			name: "los satélites reducen el préstamo", baseMB: 512, reserveMB: 6000,
			hostMB: 12000, ceilingPct: 85, hardQuotaMB: 4096, want: 2048,
		},
		{
			// 2 alumnos × 512 MB + 256 MB de MongoDB sobre un nodo de 8 GB: entra
			// con 1024 MB, que deja sitio a otro del mismo tamaño.
			name: "dos alumnos con mongo entran sin apretar", baseMB: 512, committedMB: 1024, reserveMB: 256,
			hostMB: 8192, ceilingPct: 85, hardQuotaMB: 4096, want: 2048,
		},
		{
			// El décimo alumno en un nodo de 8 GB ya no entraba con la cuenta de
			// 768 MB por laboratorio.
			name: "la cuenta vieja negaba el décimo alumno", baseMB: 768, committedMB: 9 * 768,
			hostMB: 8192, ceilingPct: 85, hardQuotaMB: 4096, want: 0,
		},
		{
			// Con la cuota sin el delta del satélite, el décimo alumno entra con
			// la base: no hay hueco para prestar.
			name: "la cuota actual admite el décimo alumno", baseMB: 512, committedMB: 9 * 512, reserveMB: 256,
			hostMB: 8192, ceilingPct: 85, hardQuotaMB: 4096, want: 1024,
		},
		{
			// El alumno 30 sobre un nodo de 18 GB, sin satélites.
			name: "treinta alumnos sin satélites", baseMB: 512, committedMB: 29 * 512,
			hostMB: 18432, ceilingPct: 85, hardQuotaMB: 4096, want: 512,
		},
		{
			name: "treinta alumnos con mongo y redis", baseMB: 512, committedMB: 29 * 512, reserveMB: 352,
			hostMB: 18432, ceilingPct: 85, hardQuotaMB: 4096, want: 0,
		},
		{
			name: "sin host no se decide", baseMB: 512,
			hostMB: 0, ceilingPct: 85, hardQuotaMB: 4096, want: 0,
		},
		{
			// Sin techo configurado rige el 85%: 8192 × 85% = 6963.
			name: "techo por defecto", baseMB: 512, committedMB: 7000,
			hostMB: 8192, ceilingPct: 0, hardQuotaMB: 4096, want: 0,
		},
		{
			// Un perfil Intensivo (Datos & IA) que pide 4096 no se recorta.
			name: "la base de 4096 se respeta", baseMB: 4096,
			hostMB: 12000, ceilingPct: 85, hardQuotaMB: 4096, want: 4096,
		},
		{
			name: "la base de 1536 dobla a 2048 y no a 4096", baseMB: 1536,
			hostMB: 8192, ceilingPct: 85, hardQuotaMB: 4096, want: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loanQuotaMB(tt.baseMB, tt.committedMB, tt.reserveMB, tt.hostMB, tt.ceilingPct, tt.hardQuotaMB)
			if got != tt.want {
				t.Fatalf("cuota esperada %d MB, obtenida %d MB", tt.want, got)
			}
		})
	}
}

// Dos aperturas seguidas no pueden quedarse con el mismo hueco: la primera
// aparta su cuota mientras no tiene fila y la segunda ya ve ese monto tomado.
func TestReserveAdmissionQuota_NoReparteElMismoHuecoDosVeces(t *testing.T) {
	svc := NewPracticeEnvironmentService(nil, nil, nil)

	first, releaseFirst, err := svc.reserveAdmissionQuota(nil, 512, 0, 12000, 85)
	if err != nil {
		t.Fatalf("la primera apertura debió entrar: %v", err)
	}
	if first != 4096 {
		t.Fatalf("la primera apertura esperaba 4096 MB, obtuvo %d MB", first)
	}

	second, releaseSecond, err := svc.reserveAdmissionQuota(nil, 512, 0, 12000, 85)
	if err != nil {
		t.Fatalf("la segunda apertura debió entrar: %v", err)
	}
	if second != 2048 {
		t.Fatalf("la segunda apertura esperaba 2048 MB sobre el hueco restante, obtuvo %d MB", second)
	}

	releaseFirst()
	releaseSecond()
	// Liberar dos veces no devuelve el monto dos veces.
	releaseFirst()

	third, releaseThird, err := svc.reserveAdmissionQuota(nil, 512, 0, 12000, 85)
	if err != nil {
		t.Fatalf("la tercera apertura debió entrar: %v", err)
	}
	if third != 4096 {
		t.Fatalf("con las reservas liberadas esperaba 4096 MB, obtuvo %d MB", third)
	}
	releaseThird()
}

// Sin hueco ni para la base, el entorno se rechaza.
func TestReserveAdmissionQuota_RechazaSinHuecoParaLaBase(t *testing.T) {
	svc := NewPracticeEnvironmentService(nil, nil, nil)
	active := []*domain.PracticeEnvironment{{ID: "ws-1", MemoryLimitMB: 512}}

	if _, _, err := svc.reserveAdmissionQuota(active, 512, 0, 1024, 85); !errors.Is(err, ErrHostMemoryExhausted) {
		t.Fatalf("esperaba ErrHostMemoryExhausted, obtuvo %v", err)
	}
}

// El préstamo no sobrevive a la hibernación: el entorno vuelve con la base si
// la clase ya llenó el nodo, y con lo que quepa si todavía hay hueco.
func TestLoanQuotaForReactivation_CaducaElPrestamo(t *testing.T) {
	svc := NewPracticeEnvironmentService(nil, nil, nil)
	svc.hostTotalMBFn = func() (int, bool) { return 12000, true }
	ctx := context.Background()
	env := &domain.PracticeEnvironment{ID: "env-a"}

	if got := svc.loanQuotaForReactivation(ctx, env, 512, nil, 85); got != 4096 {
		t.Fatalf("con el nodo vacío esperaba 4096 MB, obtuvo %d MB", got)
	}

	full := make([]*domain.PracticeEnvironment, 0, 6)
	for i := 0; i < 6; i++ {
		full = append(full, &domain.PracticeEnvironment{ID: fmt.Sprintf("env-%d", i), MemoryLimitMB: 2048})
	}
	if got := svc.loanQuotaForReactivation(ctx, env, 512, full, 85); got != 512 {
		t.Fatalf("con la clase llena esperaba la base de 512 MB, obtuvo %d MB", got)
	}

	// El propio entorno no se cuenta dos veces: al hibernar dejó de sumar.
	withPeer := []*domain.PracticeEnvironment{
		env,
		{ID: "env-b", MemoryLimitMB: 4096},
	}
	if got := svc.loanQuotaForReactivation(ctx, env, 512, withPeer, 85); got != 2048 {
		t.Fatalf("esperaba 2048 MB sobre el hueco restante, obtuvo %d MB", got)
	}
}
