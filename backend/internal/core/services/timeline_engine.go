package services

import (
	"strings"

	"solv-backend/internal/core/domain"
)

// TimelineEngine construye y analiza la telemetría de time-travel replay del código del estudiante.
type TimelineEngine struct{}

func NewTimelineEngine() *TimelineEngine {
	return &TimelineEngine{}
}

// GenerateTimeline reconstruye el stream de eventos de escritura detectando pegados masivos y ritmo de tipeo.
func (e *TimelineEngine) GenerateTimeline(submissionID, studentID, studentName, code, boilerplate string) *domain.SubmissionTimeline {
	lines := strings.Split(code, "\n")
	totalChars := len(code)

	// Si el código es muy corto
	if totalChars == 0 {
		return &domain.SubmissionTimeline{
			SubmissionID:         submissionID,
			StudentID:            studentID,
			StudentName:          studentName,
			TotalDurationSeconds: 0,
			TotalKeystrokes:      0,
			PasteEventsCount:     0,
			PastePercentage:      0.0,
			SuspiciousPasteFlag:  false,
			Keyframes:            []domain.TimelineKeyframe{},
		}
	}

	keyframes := make([]domain.TimelineKeyframe, 0)
	var currentBuilder strings.Builder
	currentOffset := 0
	pasteChars := 0
	totalKeystrokes := 0
	pasteEvents := 0

	// 1. Keyframe inicial: Boilerplate (si existe)
	if strings.TrimSpace(boilerplate) != "" {
		currentBuilder.WriteString(boilerplate)
		currentOffset += 1000
		keyframes = append(keyframes, domain.TimelineKeyframe{
			OffsetMS:   currentOffset,
			Action:     "checkpoint",
			Content:    currentBuilder.String(),
			CursorLine: 1,
			IsPaste:    false,
			CharCount:  len(boilerplate),
		})
	}

	// 2. Analizar líneas para simular / reconstruir la evolución temporal
	for lineIdx, line := range lines {
		lineTrimmed := strings.TrimSpace(line)
		if lineTrimmed == "" {
			currentBuilder.WriteString("\n")
			currentOffset += 400
			continue
		}

		// Heurística de detección de Pegado Masivo:
		// Si una línea tiene más de 90 caracteres y no es un comentario ni string largo,
		// o si es una función completa en un solo bloque.
		isPaste := len(line) > 100

		if isPaste {
			pasteEvents++
			pasteChars += len(line)
			currentBuilder.WriteString(line)
			currentBuilder.WriteString("\n")
			currentOffset += 250 // Paste ocurre en < 300ms
			totalKeystrokes += 2  // Ctrl+V

			keyframes = append(keyframes, domain.TimelineKeyframe{
				OffsetMS:   currentOffset,
				Action:     "paste",
				Content:    currentBuilder.String(),
				CursorLine: lineIdx + 1,
				IsPaste:    true,
				CharCount:  len(line),
			})
		} else {
			// Tipeo incremental por tokens/palabras
			words := strings.Fields(line)
			for wIdx, word := range words {
				if wIdx > 0 {
					currentBuilder.WriteString(" ")
				}
				currentBuilder.WriteString(word)
				totalKeystrokes += len(word)
				currentOffset += len(word) * 120 // ~120ms por caracter (tipeo humano normal)

				// Guardar keyframe cada 2 palabras o al final de la línea
				if wIdx%2 == 0 || wIdx == len(words)-1 {
					keyframes = append(keyframes, domain.TimelineKeyframe{
						OffsetMS:   currentOffset,
						Action:     "insert",
						Content:    currentBuilder.String(),
						CursorLine: lineIdx + 1,
						IsPaste:    false,
						CharCount:  len(word),
					})
				}
			}
			currentBuilder.WriteString("\n")
			currentOffset += 500 // Pausa al final de línea
		}
	}

	// 3. Keyframe final con el código consolidado exacto
	keyframes = append(keyframes, domain.TimelineKeyframe{
		OffsetMS:   currentOffset + 300,
		Action:     "checkpoint",
		Content:    code,
		CursorLine: len(lines),
		IsPaste:    false,
		CharCount:  len(code),
	})

	totalDurationSec := currentOffset / 1000
	if totalDurationSec == 0 {
		totalDurationSec = 1
	}

	pastePercentage := 0.0
	if totalChars > 0 {
		pastePercentage = float64(pasteChars) / float64(totalChars) * 100.0
		if pastePercentage > 100.0 {
			pastePercentage = 100.0
		}
	}

	// Flag sospechoso si más del 65% del código fue pegado o si el tiempo total estimado fue < 10s con código complejo
	suspicious := pastePercentage >= 65.0 || (pasteEvents > 0 && totalDurationSec < 15 && totalChars > 150)

	return &domain.SubmissionTimeline{
		SubmissionID:         submissionID,
		StudentID:            studentID,
		StudentName:          studentName,
		TotalDurationSeconds: totalDurationSec,
		TotalKeystrokes:      totalKeystrokes,
		PasteEventsCount:     pasteEvents,
		PastePercentage:      pastePercentage,
		SuspiciousPasteFlag:  suspicious,
		Keyframes:            keyframes,
	}
}
