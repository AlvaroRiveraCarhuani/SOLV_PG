package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Estado de una solicitud de entorno del docente.
const (
	EnvironmentRequestEnRevision     = "EN_REVISION"
	EnvironmentRequestAprobada       = "APROBADA"
	EnvironmentRequestNecesitaCambio = "NECESITA_CAMBIO"
	EnvironmentRequestNoViable       = "NO_VIABLE"
)

// Usos declarados por el docente. No son parámetros técnicos: describen qué
// harán los estudiantes y el backend los traduce a la modalidad del entorno.
const (
	EnvironmentUsePractice = "practice" // practicar en clase, con editor
	EnvironmentUseGrade    = "grade"    // entregar y calificar automáticamente
)

var (
	ErrEnvironmentRequestNotFound  = errors.New("solicitud de entorno no encontrada")
	ErrEnvironmentRequestDuplicate = errors.New("ya existe una solicitud con la misma receta")
	ErrEnvironmentRequestInvalid   = errors.New("solicitud de entorno inválida")
)

// EnvironmentRequestSubject es una materia que espera el entorno. Se acumulan
// cuando varios docentes piden la misma receta.
type EnvironmentRequestSubject struct {
	SubjectID     string    `json:"subject_id,omitempty"`
	SubjectName   string    `json:"subject_name"`
	RequestedBy   string    `json:"requested_by,omitempty"`
	RequesterName string    `json:"requester_name,omitempty"`
	AddedAt       time.Time `json:"added_at"`
}

// EnvironmentRequestComponent es la foto de una pieza del catálogo al momento
// de pedir: el docente ve el nombre, el administrador ve la pieza resuelta.
type EnvironmentRequestComponent struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Layer     string `json:"layer"` // runtime | addon | database
	IconKey   string `json:"icon_key,omitempty"`
	BaseImage string `json:"base_image,omitempty"`
}

// EnvironmentRequestMissingComponent es una tecnología que el docente nombró y
// el catálogo no tiene. Se guarda para que el administrador decida si la crea.
type EnvironmentRequestMissingComponent struct {
	Name     string `json:"name"`
	Layer    string `json:"layer"`
	Category string `json:"category,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// EnvironmentRequest es la solicitud tal como se persiste y se muestra.
type EnvironmentRequest struct {
	ID                 string                               `json:"id"`
	TenantID           string                               `json:"tenant_id"`
	RequestedBy        string                               `json:"requested_by"`
	RequesterName      string                               `json:"requester_name"`
	Title              string                               `json:"title"`
	Description        string                               `json:"description"`
	TeacherText        string                               `json:"teacher_text"`
	Uses               []string                             `json:"uses"`
	TargetEnvironment  string                               `json:"target_environment"`
	StackConfig        StackConfig                          `json:"stack_config"`
	Components         []EnvironmentRequestComponent        `json:"components"`
	MissingComponents  []EnvironmentRequestMissingComponent `json:"missing_components"`
	Subjects           []EnvironmentRequestSubject          `json:"subjects"`
	StackSignature     string                               `json:"stack_signature"`
	Status             string                               `json:"status"`
	ReviewNote         string                               `json:"review_note"`
	ReviewedBy         string                               `json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time                           `json:"reviewed_at,omitempty"`
	ResolvedTemplateID string                               `json:"resolved_template_id,omitempty"`
	CreatedAt          time.Time                            `json:"created_at"`
	UpdatedAt          time.Time                            `json:"updated_at"`
}

// CreateEnvironmentRequestDTO es lo que envía el docente. Todo lo técnico es
// opcional: si no elige piezas, la descripción alcanza para que el
// administrador arme el entorno.
type CreateEnvironmentRequestDTO struct {
	Title             string                               `json:"title"`
	Description       string                               `json:"description"`
	TeacherText       string                               `json:"teacher_text"`
	Uses              []string                             `json:"uses"`
	StackConfig       StackConfig                          `json:"stack_config"`
	MissingComponents []EnvironmentRequestMissingComponent `json:"missing_components,omitempty"`
	SubjectID         string                               `json:"subject_id,omitempty"`
	SubjectName       string                               `json:"subject_name,omitempty"`
}

// EnvironmentRequestPreviewDTO pide el armado de una receta sin guardarla.
type EnvironmentRequestPreviewDTO struct {
	Uses        []string    `json:"uses"`
	StackConfig StackConfig `json:"stack_config"`
}

// EnvironmentRequestPreviewComponent describe una pieza resuelta para el panel
// de resumen que el docente revisa antes de enviar.
type EnvironmentRequestPreviewComponent struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Layer     string `json:"layer"`
	IconKey   string `json:"icon_key,omitempty"`
	BaseImage string `json:"base_image,omitempty"`
	Missing   bool   `json:"missing"`
}

// EnvironmentRequestTemplateRef apunta a una plantilla ya aprobada con la misma
// receta, para ofrecerla en lugar de crear otra igual.
type EnvironmentRequestTemplateRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EnvironmentRequestDuplicate es una solicitud en revisión con la misma huella.
type EnvironmentRequestDuplicate struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	Subjects      []string `json:"subjects"`
	RequestedByMe bool     `json:"requested_by_me"`
}

// EnvironmentRequestPreview es el resultado de armar la receta.
type EnvironmentRequestPreview struct {
	SuggestedName     string                               `json:"suggested_name"`
	TargetEnvironment string                               `json:"target_environment"`
	RecommendedRAMMB  int                                  `json:"recommended_ram_mb"`
	Signature         string                               `json:"signature"`
	Components        []EnvironmentRequestPreviewComponent `json:"components"`
	MissingComponents []EnvironmentRequestMissingComponent `json:"missing_components"`
	DuplicateRequest  *EnvironmentRequestDuplicate         `json:"duplicate_request,omitempty"`
	ExistingTemplate  *EnvironmentRequestTemplateRef       `json:"existing_template,omitempty"`
}

// EnvironmentRequestCreated es la respuesta del envío: si se sumó a una
// solicitud existente, `Joined` lo dice y `Request` trae la ficha completa.
type EnvironmentRequestCreated struct {
	Request *EnvironmentRequest `json:"request"`
	Joined  bool                `json:"joined"`
	Message string              `json:"message"`
}

// ReviewEnvironmentRequestDTO es la respuesta del administrador.
type ReviewEnvironmentRequestDTO struct {
	Status             string `json:"status"`
	ReviewNote         string `json:"review_note"`
	ResolvedTemplateID string `json:"resolved_template_id,omitempty"`
}

// DeriveTargetEnvironment traduce el uso declarado por el docente a la
// modalidad del entorno. Sin uso declarado se asume práctica, que es el caso
// más frecuente y el menos restrictivo.
func DeriveTargetEnvironment(uses []string) string {
	needsPractice := false
	needsGrade := false
	for _, use := range uses {
		switch strings.ToLower(strings.TrimSpace(use)) {
		case EnvironmentUsePractice:
			needsPractice = true
		case EnvironmentUseGrade:
			needsGrade = true
		}
	}
	switch {
	case needsPractice && needsGrade:
		return TargetEnvironmentAmbos
	case needsGrade:
		return TargetEnvironmentJuezEfimero
	default:
		return TargetEnvironmentIDEPersistente
	}
}

// NormalizeEnvironmentUses limpia y deduplica los usos declarados.
func NormalizeEnvironmentUses(uses []string) []string {
	seen := make(map[string]struct{}, len(uses))
	result := make([]string, 0, len(uses))
	for _, use := range uses {
		clean := strings.ToLower(strings.TrimSpace(use))
		if clean != EnvironmentUsePractice && clean != EnvironmentUseGrade {
			continue
		}
		if _, dup := seen[clean]; dup {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	if len(result) == 0 {
		result = append(result, EnvironmentUsePractice)
	}
	return result
}

// EnvironmentRequestRepository persiste las solicitudes del docente.
type EnvironmentRequestRepository interface {
	Create(ctx context.Context, req *EnvironmentRequest) (*EnvironmentRequest, error)
	FindPendingBySignature(ctx context.Context, tenantID, signature string) (*EnvironmentRequest, error)
	AddSubject(ctx context.Context, requestID string, subject EnvironmentRequestSubject) error
	ListMine(ctx context.Context, tenantID, userID string) ([]*EnvironmentRequest, error)
	GetByID(ctx context.Context, tenantID, id string) (*EnvironmentRequest, error)
	ListForReview(ctx context.Context, tenantID, status string) ([]*EnvironmentRequest, error)
	Review(ctx context.Context, tenantID, id, reviewerID string, dto ReviewEnvironmentRequestDTO) (*EnvironmentRequest, error)
	FindApprovedTemplateByImage(ctx context.Context, tenantID, image string) (*EnvironmentRequestTemplateRef, error)
}
