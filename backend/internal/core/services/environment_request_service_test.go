package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestDeriveTargetEnvironmentFromUses(t *testing.T) {
	cases := []struct {
		name string
		uses []string
		want string
	}{
		{"sin usos declarados asume práctica", nil, domain.TargetEnvironmentIDEPersistente},
		{"practicar", []string{"practice"}, domain.TargetEnvironmentIDEPersistente},
		{"calificar", []string{"grade"}, domain.TargetEnvironmentJuezEfimero},
		{"ambos", []string{"grade", "practice"}, domain.TargetEnvironmentAmbos},
		{"uso desconocido se ignora", []string{"no-existe"}, domain.TargetEnvironmentIDEPersistente},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.DeriveTargetEnvironment(tc.uses); got != tc.want {
				t.Fatalf("DeriveTargetEnvironment(%v) = %q, esperaba %q", tc.uses, got, tc.want)
			}
		})
	}
}

func TestNormalizeEnvironmentUsesDedupesAndDefaults(t *testing.T) {
	got := domain.NormalizeEnvironmentUses([]string{" GRADe ", "grade", "practice", "otro"})
	if len(got) != 2 || got[0] != "grade" || got[1] != "practice" {
		t.Fatalf("usos normalizados inesperados: %#v", got)
	}
	if fallback := domain.NormalizeEnvironmentUses(nil); len(fallback) != 1 || fallback[0] != "practice" {
		t.Fatalf("sin usos debe quedar práctica, obtuve %#v", fallback)
	}
}

func TestCreateRejectsShortDescription(t *testing.T) {
	svc := NewEnvironmentRequestService(&fakeEnvironmentRequestRepo{}, nil)
	_, err := svc.Create(context.Background(), "tenant", "teacher", domain.CreateEnvironmentRequestDTO{
		Description: "corto",
	})
	if !errors.Is(err, domain.ErrEnvironmentRequestInvalid) {
		t.Fatalf("esperaba ErrEnvironmentRequestInvalid, obtuve %v", err)
	}
}

func TestReviewRequiresTemplateToApprove(t *testing.T) {
	svc := NewEnvironmentRequestService(&fakeEnvironmentRequestRepo{}, nil)
	_, err := svc.Review(context.Background(), "tenant", "req-1", "admin", domain.ReviewEnvironmentRequestDTO{
		Status: domain.EnvironmentRequestAprobada,
	})
	if !errors.Is(err, domain.ErrEnvironmentRequestInvalid) {
		t.Fatalf("aprobar sin plantilla debe fallar, obtuve %v", err)
	}

	_, err = svc.Review(context.Background(), "tenant", "req-1", "admin", domain.ReviewEnvironmentRequestDTO{
		Status:     domain.EnvironmentRequestEnRevision,
		ReviewNote: "cualquiera",
	})
	if !errors.Is(err, domain.ErrEnvironmentRequestInvalid) {
		t.Fatalf("un estado fuera de la revisión debe fallar, obtuve %v", err)
	}
}

// Un docente que pide la misma receta que otra solicitud en revisión no abre una
// ficha paralela: suma su materia a la que ya existe.
func TestCreateJoinsExistingRequestWithSameSignature(t *testing.T) {
	compRepo := &fakeComponentRepo{comps: map[string]*domain.TemplateComponent{
		"python-312": {
			ID: "python-312", Slug: "python-312", Name: "Python 3.12", Layer: domain.ComponentLayerRuntime,
			BaseImage: "python:3.12-slim-bookworm", IconKey: "python",
			ToolsProvided: []string{"python3", "pip"},
		},
		"node-nestjs": {
			ID: "node-nestjs", Slug: "node-nestjs", Name: "NestJS CLI", Layer: domain.ComponentLayerAddon,
			SubCategory: "framework", IconKey: "nestjs", InstallMode: "npm",
			BuildLayerSnippet: "RUN npm install -g @nestjs/cli",
			ToolsProvided:     []string{"nest"},
		},
	}}
	repo := &fakeEnvironmentRequestRepo{}
	svc := NewEnvironmentRequestService(repo, compRepo)

	first, err := svc.Create(context.Background(), "tenant", "teacher-a", domain.CreateEnvironmentRequestDTO{
		Description: "Los estudiantes construyen una API con NestJS.",
		Uses:        []string{"practice", "grade"},
		StackConfig: domain.StackConfig{RuntimeID: "python-312", AddonIDs: []string{"node-nestjs"}},
		SubjectID:   "subject-db",
		SubjectName: "Bases de Datos",
	})
	if err != nil {
		t.Fatalf("primera solicitud: %v", err)
	}
	if first.Joined {
		t.Fatal("la primera solicitud no debería sumarse a nada")
	}
	if first.Request.TargetEnvironment != domain.TargetEnvironmentAmbos {
		t.Fatalf("esperaba modalidad AMBOS, obtuve %q", first.Request.TargetEnvironment)
	}
	if first.Request.StackSignature == "" {
		t.Fatal("la solicitud con piezas debe tener huella")
	}

	second, err := svc.Create(context.Background(), "tenant", "teacher-b", domain.CreateEnvironmentRequestDTO{
		Description: "Quiero lo mismo pero para mi curso de arquitectura de software.",
		Uses:        []string{"practice", "grade"},
		StackConfig: domain.StackConfig{RuntimeID: "python-312", AddonIDs: []string{"node-nestjs"}},
		SubjectID:   "subject-arch",
		SubjectName: "Arquitectura de Software",
	})
	if err != nil {
		t.Fatalf("segunda solicitud: %v", err)
	}
	if !second.Joined {
		t.Fatal("la segunda solicitud con la misma receta debería sumarse")
	}
	if second.Request.ID != first.Request.ID {
		t.Fatalf("esperaba la misma ficha %s, obtuve %s", first.Request.ID, second.Request.ID)
	}
	if len(repo.requests) != 1 {
		t.Fatalf("solo debía existir una solicitud, hay %d", len(repo.requests))
	}
	if len(second.Request.Subjects) != 2 {
		t.Fatalf("esperaba dos materias en la ficha, hay %d", len(second.Request.Subjects))
	}
}

// La huella no depende del orden en que el docente eligió las piezas.
func TestSignatureStableAcrossAddonOrder(t *testing.T) {
	compRepo := &fakeComponentRepo{comps: map[string]*domain.TemplateComponent{
		"python-312": {ID: "python-312", Slug: "python-312", Name: "Python 3.12", Layer: domain.ComponentLayerRuntime, BaseImage: "python:3.12-slim-bookworm"},
		"a":          {ID: "a", Slug: "a", Name: "A", Layer: domain.ComponentLayerAddon, BuildLayerSnippet: "RUN a"},
		"b":          {ID: "b", Slug: "b", Name: "B", Layer: domain.ComponentLayerAddon, BuildLayerSnippet: "RUN b"},
	}}
	svc := NewEnvironmentRequestService(&fakeEnvironmentRequestRepo{}, compRepo)

	one, err := svc.Preview(context.Background(), "tenant", "teacher", domain.EnvironmentRequestPreviewDTO{
		Uses:        []string{"practice"},
		StackConfig: domain.StackConfig{RuntimeID: "python-312", AddonIDs: []string{"a", "b"}},
	})
	if err != nil {
		t.Fatalf("preview uno: %v", err)
	}
	two, err := svc.Preview(context.Background(), "tenant", "teacher", domain.EnvironmentRequestPreviewDTO{
		Uses:        []string{"practice"},
		StackConfig: domain.StackConfig{RuntimeID: "python-312", AddonIDs: []string{"b", "a"}},
	})
	if err != nil {
		t.Fatalf("preview dos: %v", err)
	}
	if one.Signature == "" || one.Signature != two.Signature {
		t.Fatalf("la huella debe ser estable: %q vs %q", one.Signature, two.Signature)
	}
}

// El docente ve nombres legibles, nunca slugs ni imágenes.
func TestPreviewExposesHumanNamesAndFlagsMissing(t *testing.T) {
	compRepo := &fakeComponentRepo{comps: map[string]*domain.TemplateComponent{
		"python-312": {ID: "python-312", Slug: "python-312", Name: "Python 3.12", Layer: domain.ComponentLayerRuntime, BaseImage: "python:3.12-slim-bookworm"},
	}}
	svc := NewEnvironmentRequestService(&fakeEnvironmentRequestRepo{}, compRepo)

	preview, err := svc.Preview(context.Background(), "tenant", "teacher", domain.EnvironmentRequestPreviewDTO{
		StackConfig: domain.StackConfig{RuntimeID: "python-312", AddonIDs: []string{"graphql-kit"}},
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	var sawRuntime, sawMissing bool
	for _, c := range preview.Components {
		if c.Slug == "python-312" && c.Name == "Python 3.12" && !c.Missing {
			sawRuntime = true
		}
		if c.Slug == "graphql-kit" && c.Missing {
			sawMissing = true
		}
	}
	if !sawRuntime || !sawMissing {
		t.Fatalf("esperaba runtime resuelto y pieza faltante, obtuve %#v", preview.Components)
	}
	if strings.Contains(preview.SuggestedName, "python:3.12") {
		t.Fatalf("el nombre sugerido no debe exponer la imagen: %q", preview.SuggestedName)
	}
}

// --- dobles de prueba -------------------------------------------------------

type fakeComponentRepo struct {
	domain.TemplateComponentRepository
	comps map[string]*domain.TemplateComponent
}

func (f *fakeComponentRepo) GetTemplateComponentByIDOrSlug(_ context.Context, _, idOrSlug string) (*domain.TemplateComponent, error) {
	if comp, ok := f.comps[idOrSlug]; ok {
		return comp, nil
	}
	return nil, errors.New("componente no encontrado")
}

type fakeEnvironmentRequestRepo struct {
	domain.EnvironmentRequestRepository
	requests []*domain.EnvironmentRequest
}

func (f *fakeEnvironmentRequestRepo) Create(_ context.Context, req *domain.EnvironmentRequest) (*domain.EnvironmentRequest, error) {
	req.ID = "req-" + string(rune('a'+len(f.requests)))
	req.CreatedAt = time.Now()
	req.UpdatedAt = req.CreatedAt
	f.requests = append(f.requests, req)
	return req, nil
}

func (f *fakeEnvironmentRequestRepo) FindPendingBySignature(_ context.Context, _, signature string) (*domain.EnvironmentRequest, error) {
	if signature == "" {
		return nil, nil
	}
	for _, req := range f.requests {
		if req.StackSignature == signature && req.Status == domain.EnvironmentRequestEnRevision {
			return req, nil
		}
	}
	return nil, nil
}

func (f *fakeEnvironmentRequestRepo) AddSubject(_ context.Context, requestID string, subject domain.EnvironmentRequestSubject) error {
	for _, req := range f.requests {
		if req.ID != requestID {
			continue
		}
		for _, existing := range req.Subjects {
			if existing.SubjectID == subject.SubjectID {
				return nil
			}
		}
		req.Subjects = append(req.Subjects, subject)
	}
	return nil
}

func (f *fakeEnvironmentRequestRepo) GetByID(_ context.Context, _, id string) (*domain.EnvironmentRequest, error) {
	for _, req := range f.requests {
		if req.ID == id {
			return req, nil
		}
	}
	return nil, domain.ErrEnvironmentRequestNotFound
}

func (f *fakeEnvironmentRequestRepo) ListMine(context.Context, string, string) ([]*domain.EnvironmentRequest, error) {
	return f.requests, nil
}

func (f *fakeEnvironmentRequestRepo) ListForReview(context.Context, string, string) ([]*domain.EnvironmentRequest, error) {
	return f.requests, nil
}

func (f *fakeEnvironmentRequestRepo) Review(_ context.Context, _, id, reviewerID string, dto domain.ReviewEnvironmentRequestDTO) (*domain.EnvironmentRequest, error) {
	for _, req := range f.requests {
		if req.ID == id {
			req.Status = dto.Status
			req.ReviewNote = dto.ReviewNote
			req.ReviewedBy = reviewerID
			return req, nil
		}
	}
	return nil, domain.ErrEnvironmentRequestNotFound
}

func (f *fakeEnvironmentRequestRepo) FindApprovedTemplateByImage(context.Context, string, string) (*domain.EnvironmentRequestTemplateRef, error) {
	return nil, nil
}
