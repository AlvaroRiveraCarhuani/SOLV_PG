package postgres

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestApplyComponentUsageCounts(t *testing.T) {
	items := []*domain.TemplateComponent{
		{ID: "uuid-python", Slug: "python-312"},
		{ID: "uuid-fastapi", Slug: "fastapi"},
		{ID: "uuid-unused", Slug: "unused"},
		nil,
	}

	// El compositor acepta id o slug, así que un mismo componente puede aparecer
	// referenciado de las dos formas en plantillas distintas.
	applyComponentUsageCounts(items, map[string]int{
		"uuid-python": 2,
		"python-312":  1,
		"fastapi":     3,
	})

	if items[0].UsageCount != 3 {
		t.Errorf("esperado 3 usos para python (2 por id + 1 por slug), obtenido %d", items[0].UsageCount)
	}
	if items[1].UsageCount != 3 {
		t.Errorf("esperado 3 usos por slug para fastapi, obtenido %d", items[1].UsageCount)
	}
	if items[2].UsageCount != 0 {
		t.Errorf("esperado 0 usos para una tecnologia sin referencias, obtenido %d", items[2].UsageCount)
	}
}

func TestApplyComponentUsageCounts_SinReferencias(t *testing.T) {
	items := []*domain.TemplateComponent{{ID: "uuid-a", Slug: "a"}}

	applyComponentUsageCounts(items, nil)

	if items[0].UsageCount != 0 {
		t.Errorf("esperado 0 usos sin mapa de referencias, obtenido %d", items[0].UsageCount)
	}
}
