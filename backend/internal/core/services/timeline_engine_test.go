package services

import (
	"testing"
)

func TestTimelineEngine_TypingAndPasteDetection(t *testing.T) {
	engine := NewTimelineEngine()

	// 1. Código con tipeo incremental
	codeTyped := `
def sumar(a, b):
    return a + b
`
	timelineTyped := engine.GenerateTimeline("sub-1", "stu-1", "Carlos", codeTyped, "")

	if len(timelineTyped.Keyframes) == 0 {
		t.Fatalf("Esperados keyframes en el timeline, obtenido 0")
	}

	if timelineTyped.SuspiciousPasteFlag {
		t.Errorf("Código tipeado no debería marcarse como sospechoso")
	}

	// 2. Código con pegado masivo de función en una sola línea larga o bloque
	longSnippet := "def solve_complex_graph_dijkstra_shortest_path(nodes, edges, start_node, target_node, heuristic_weight_matrix): return [min(edges)] * len(nodes)"
	timelinePaste := engine.GenerateTimeline("sub-2", "stu-2", "Ana", longSnippet, "")

	if timelinePaste.PasteEventsCount == 0 {
		t.Errorf("Esperado al menos 1 evento de pegado masivo detectado")
	}

	if timelinePaste.PastePercentage < 50.0 {
		t.Errorf("Esperado porcentaje de pegado >= 50%%, obtenido: %.2f%%", timelinePaste.PastePercentage)
	}
}
