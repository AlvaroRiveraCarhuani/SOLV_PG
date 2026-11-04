package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
)

type SubjectService struct {
	repo domain.SubjectRepository
}

func NewSubjectService(repo domain.SubjectRepository) *SubjectService {
	return &SubjectService{repo: repo}
}

// NewSubjectInput reúne el alta de una materia. Existe para que el formulario
// pueda crecer —como creció con la sección— sin alargar la firma del servicio con
// parámetros posicionales que después nadie recuerda en qué orden van.
type NewSubjectInput struct {
	Name              string
	Code              string
	Section           string
	TeacherID         *string
	AcademicPeriodID  *string
	ClassroomCourseID *string
	TemplateID        *string
}

func (s *SubjectService) CreateSubject(ctx context.Context, tenantID, name, code string, classroomCourseID *string) (*domain.Subject, error) {
	return s.Create(ctx, tenantID, NewSubjectInput{Name: name, Code: code, ClassroomCourseID: classroomCourseID})
}

func (s *SubjectService) CreateSubjectWithDetails(ctx context.Context, tenantID, name, code string, teacherID, academicPeriodID, classroomCourseID, templateID *string) (*domain.Subject, error) {
	return s.Create(ctx, tenantID, NewSubjectInput{
		Name:              name,
		Code:              code,
		TeacherID:         teacherID,
		AcademicPeriodID:  academicPeriodID,
		ClassroomCourseID: classroomCourseID,
		TemplateID:        templateID,
	})
}

func (s *SubjectService) Create(ctx context.Context, tenantID string, in NewSubjectInput) (*domain.Subject, error) {
	if in.Name == "" || in.Code == "" {
		return nil, errors.New("name and code are required")
	}
	section, err := normalizeSection(in.Section)
	if err != nil {
		return nil, err
	}
	if in.TemplateID != nil && *in.TemplateID != "" {
		status, err := s.repo.GetTemplateStatus(ctx, *in.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("la plantilla de laboratorio especificada no existe: %w", err)
		}
		if status != "approved" && status != "APROBADA" {
			return nil, fmt.Errorf("la plantilla seleccionada no ha sido aprobada por la auditoría de seguridad (estado actual: %s)", status)
		}
	}

	subject := &domain.Subject{
		ID:                uuid.New().String(),
		TenantID:          tenantID,
		Name:              in.Name,
		Code:              strings.ToUpper(strings.TrimSpace(in.Code)),
		TeacherID:         in.TeacherID,
		AcademicPeriodID:  in.AcademicPeriodID,
		ClassroomCourseID: in.ClassroomCourseID,
		TemplateID:        in.TemplateID,
	}
	if section != "" {
		subject.Section = &section
	}

	if err := s.repo.Create(ctx, subject); err != nil {
		return nil, fmt.Errorf("failed to create subject: %w", err)
	}
	return subject, nil
}

// normalizeSection limita la sección a letras y números. Viaja dentro del código
// del curso ("SIS-211-1"), así que un espacio o una barra lo dejarían ilegible y
// romperían la lectura del código en las tarjetas y en el aula.
func normalizeSection(raw string) (string, error) {
	section := strings.ToUpper(strings.TrimSpace(raw))
	if section == "" {
		return "", nil
	}
	if len(section) > 20 {
		return "", errors.New("la sección no puede superar los 20 caracteres")
	}
	for _, r := range section {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return "", errors.New("la sección solo admite letras y números")
		}
	}
	return section, nil
}

func (s *SubjectService) ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error {
	return s.repo.ArchiveSubject(ctx, tenantID, subjectID, isArchived)
}

func (s *SubjectService) UpdateSubject(ctx context.Context, tenantID, subjectID, name, code string) error {
	if name == "" || code == "" {
		return errors.New("name and code are required")
	}
	return s.repo.Update(ctx, tenantID, subjectID, name, code)
}

func (s *SubjectService) ListSubjects(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *SubjectService) EnrollStudent(ctx context.Context, tenantID, studentID, subjectID string) (*domain.Enrollment, error) {
	if studentID == "" || subjectID == "" {
		return nil, errors.New("studentID and subjectID are required")
	}
	enrollment := &domain.Enrollment{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		StudentID: studentID,
		SubjectID: subjectID,
	}
	if err := s.repo.EnrollStudent(ctx, enrollment); err != nil {
		return nil, fmt.Errorf("failed to enroll student: %w", err)
	}
	return enrollment, nil
}

func (s *SubjectService) ListStudents(ctx context.Context, tenantID, subjectID string) ([]string, error) {
	return s.repo.ListStudentsBySubject(ctx, tenantID, subjectID)
}
