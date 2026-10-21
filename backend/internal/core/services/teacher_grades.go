package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

func (s *TeacherService) GetCourseGradesMatrix(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseGradesMatrix, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}
	return s.repo.GetCourseGradesMatrix(ctx, tenantID, teacherID, subjectID)
}

func (s *TeacherService) ExportCourseGradesCSV(ctx context.Context, tenantID, teacherID, subjectID string) ([]byte, string, error) {

	if tenantID == "" {
		return nil, "", domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, "", domain.ErrNotFound
	}

	matrix, err := s.repo.GetCourseGradesMatrix(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	// UTF-8 BOM para compatibilidad con Microsoft Excel
	buf.WriteString("\xef\xbb\xbf")

	writer := csv.NewWriter(&buf)

	// Encabezado
	header := []string{"ID Estudiante", "Nombre Completo", "Email"}
	for _, ex := range matrix.Exercises {
		header = append(header, ex.Title)
	}
	header = append(header, "Promedio Final")

	if err := writer.Write(header); err != nil {
		return nil, "", fmt.Errorf("error al escribir encabezado CSV: %w", err)
	}

	// Filas de estudiantes
	for _, stu := range matrix.Students {
		row := []string{stu.StudentID, stu.StudentName, stu.StudentEmail}
		for _, ex := range matrix.Exercises {
			score := stu.Grades[ex.ID]
			row = append(row, fmt.Sprintf("%d", score))
		}
		row = append(row, fmt.Sprintf("%.2f", stu.Average))

		if err := writer.Write(row); err != nil {
			return nil, "", fmt.Errorf("error al escribir fila de estudiante en CSV: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", fmt.Errorf("error al generar buffer CSV: %w", err)
	}

	codeSanitized := strings.ReplaceAll(matrix.SubjectCode, " ", "_")
	if codeSanitized == "" {
		codeSanitized = "curso"
	}
	filename := fmt.Sprintf("calificaciones_%s_%s.csv", codeSanitized, time.Now().Format("20060102"))

	return buf.Bytes(), filename, nil
}
