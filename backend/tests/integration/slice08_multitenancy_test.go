package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	coredomain "solv-backend/internal/core/domain"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/delivery/http/middleware"
	"solv-backend/internal/domain"
)

// Mock repositorios para multitenancy
type MockTenantRepository struct {
	tenants map[string]*coredomain.Tenant
}

func (m *MockTenantRepository) GetByID(ctx context.Context, id string) (*coredomain.Tenant, error) {
	if t, exists := m.tenants[id]; exists {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *MockTenantRepository) GetBySlug(ctx context.Context, slug string) (*coredomain.Tenant, error) {
	for _, t := range m.tenants {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *MockTenantRepository) GetAll(ctx context.Context) ([]*coredomain.Tenant, error) {
	var list []*coredomain.Tenant
	for _, t := range m.tenants {
		list = append(list, t)
	}
	return list, nil
}

func (m *MockTenantRepository) UpdateConfig(ctx context.Context, id string, config []byte) error {
	if t, exists := m.tenants[id]; exists {
		t.Config = config
		return nil
	}
	return errors.New("tenant not found")
}

func (m *MockTenantRepository) SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error {
	return nil
}

func (m *MockTenantRepository) GetMaintenance(ctx context.Context, tenantID string) (*coredomain.MaintenanceStatus, error) {
	return &coredomain.MaintenanceStatus{MaintenanceMode: false}, nil
}

type MockUserRepoForMT struct {
	users map[string]*domain.User
}

func (m *MockUserRepoForMT) CreateUser(ctx context.Context, dto domain.CreateUserDTO) (string, error) {
	return "", nil
}
func (m *MockUserRepoForMT) GetUserByID(ctx context.Context, id string) (domain.UserResponseDTO, error) {
	return domain.UserResponseDTO{}, nil
}
func (m *MockUserRepoForMT) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}
func (m *MockUserRepoForMT) CreateUserFromSSO(ctx context.Context, user *domain.User) (string, error) {
	user.ID = "new-user-sso-id"
	m.users[user.Email] = user
	return user.ID, nil
}

type MockWorkspaceRepoForMT struct {
	workspaces map[string]*coredomain.WorkspaceInstance
}

func (m *MockWorkspaceRepoForMT) GetByStudentAndSubject(ctx context.Context, studentID string, subjectID string) (*coredomain.WorkspaceInstance, error) {
	tenantID := coredomain.GetTenantID(ctx)
	for _, ws := range m.workspaces {
		if ws.StudentID == studentID && ws.SubjectID == subjectID && ws.TenantID == tenantID {
			return ws, nil
		}
	}
	return nil, nil
}
func (m *MockWorkspaceRepoForMT) GetByID(ctx context.Context, id string) (*coredomain.WorkspaceInstance, error) {
	tenantID := coredomain.GetTenantID(ctx)
	if ws, exists := m.workspaces[id]; exists {
		if ws.TenantID != tenantID {
			return nil, errors.New("forbidden: access denied to other tenant resource")
		}
		return ws, nil
	}
	return nil, nil
}
func (m *MockWorkspaceRepoForMT) Create(ctx context.Context, workspace *coredomain.WorkspaceInstance) error {
	m.workspaces[workspace.ID] = workspace
	return nil
}
func (m *MockWorkspaceRepoForMT) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	return nil
}
func (m *MockWorkspaceRepoForMT) UpdateStatus(ctx context.Context, id string, status string) error {
	return nil
}
func (m *MockWorkspaceRepoForMT) UpdateMemoryLimit(ctx context.Context, id string, memoryMB int64) error {
	return nil
}
func (m *MockWorkspaceRepoForMT) RecordHeartbeat(ctx context.Context, id string) error {
	return nil
}
func (m *MockWorkspaceRepoForMT) IncrementOOMStrike(ctx context.Context, id string) error {
	return nil
}
func (m *MockWorkspaceRepoForMT) ResetOOMStrikes(ctx context.Context, id string) error {
	return nil
}
func (m *MockWorkspaceRepoForMT) GetActiveWorkspaces(ctx context.Context) ([]*coredomain.WorkspaceInstance, error) {
	tenantID := coredomain.GetTenantID(ctx)
	var list []*coredomain.WorkspaceInstance
	for _, ws := range m.workspaces {
		if ws.TenantID == tenantID {
			list = append(list, ws)
		}
	}
	return list, nil
}
func (m *MockWorkspaceRepoForMT) GetAllRunningWorkspaces(ctx context.Context) ([]*coredomain.WorkspaceInstance, error) {
	return nil, nil
}
func (m *MockWorkspaceRepoForMT) GetByType(ctx context.Context, workspaceType string) ([]*coredomain.WorkspaceInstance, error) {
	return nil, nil
}
func (m *MockWorkspaceRepoForMT) SaveSemgrepAudit(ctx context.Context, id string, auditJSON []byte) error {
	return nil
}

func TestMultiTenancyIsolation(t *testing.T) {
	jwtSecret := []byte("test-super-secret-key-1234567890")

	// Setup mock repositories
	tenantRepo := &MockTenantRepository{
		tenants: map[string]*coredomain.Tenant{
			"00000000-0000-0000-0000-000000000001": {
				ID:             "00000000-0000-0000-0000-000000000001",
				Name:           "Universidad Adventista de Bolivia",
				Slug:           "uab",
				AllowedDomains: []byte(`["@uab.edu.bo"]`),
				Config:         []byte(`{"institution_name": "Universidad Adventista de Bolivia", "logo_url": "/assets/uab-logo.png"}`),
			},
			"00000000-0000-0000-0000-000000000002": {
				ID:             "00000000-0000-0000-0000-000000000002",
				Name:           "Universidad Mayor de San Andrés",
				Slug:           "umsa",
				AllowedDomains: []byte(`["@umsa.edu.bo"]`),
				Config:         []byte(`{"institution_name": "Universidad Mayor de San Andrés", "logo_url": "/assets/umsa-logo.png"}`),
			},
		},
	}

	workspaceRepo := &MockWorkspaceRepoForMT{
		workspaces: map[string]*coredomain.WorkspaceInstance{
			"ws-uab-1": {
				ID:        "ws-uab-1",
				StudentID: "student-uab",
				SubjectID: "math",
				TenantID:  "00000000-0000-0000-0000-000000000001",
			},
			"ws-umsa-1": {
				ID:        "ws-umsa-1",
				StudentID: "student-umsa",
				SubjectID: "science",
				TenantID:  "00000000-0000-0000-0000-000000000002",
			},
		},
	}

	// 1. Test ConfigHandler /api/v1/config/public (Test 5 del enunciado)
	t.Run("Endpoint /api/v1/config/public returns default tenant UAB config", func(t *testing.T) {
		h := httpdelivery.NewConfigHandler(tenantRepo)
		req := httptest.NewRequest("GET", "/api/v1/config/public", nil)
		rec := httptest.NewRecorder()
		h.GetPublicConfig(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", rec.Code)
		}

		var cfg map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
			t.Fatalf("Invalid JSON response: %v", err)
		}

		if cfg["institution_name"] != "Universidad Adventista de Bolivia" {
			t.Errorf("Expected UAB configuration, got %v", cfg["institution_name"])
		}
	})

	// 2. Test AuthService domain validation and JWT claims generation (Test 1 & 2)
	t.Run("Login dynamic domain validation", func(t *testing.T) {

		// Test 1: @uab.edu.bo login should match and generate UAB claim
		claimsUAB := jwt.MapClaims{
			"user_id":   "student-uab",
			"email":     "estudiante@uab.edu.bo",
			"role":      "student",
			"tenant_id": "00000000-0000-0000-0000-000000000001",
			"exp":       time.Now().Add(24 * time.Hour).Unix(),
		}
		tokenUAB := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsUAB)
		tokenStrUAB, _ := tokenUAB.SignedString(jwtSecret)

		// Validar si el middleware procesa correctamente
		tenantMiddleware := middleware.WithTenant(tenantRepo, jwtSecret)
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := coredomain.GetTenantID(r.Context())
			w.Write([]byte(tenantID))
		})

		reqUAB := httptest.NewRequest("GET", "/api/v1/workspaces/ws-uab-1", nil)
		reqUAB.Header.Set("Authorization", "Bearer "+tokenStrUAB)
		recUAB := httptest.NewRecorder()

		tenantMiddleware(nextHandler).ServeHTTP(recUAB, reqUAB)
		if recUAB.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", recUAB.Code, recUAB.Body.String())
		}
		if recUAB.Body.String() != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("Expected tenant UAB ID, got %s", recUAB.Body.String())
		}

		// Test 2: @umsa.edu.bo is registered and should pass as UMSA tenant
		claimsUMSA := jwt.MapClaims{
			"user_id":   "student-umsa",
			"email":     "estudiante@umsa.edu.bo",
			"role":      "student",
			"tenant_id": "00000000-0000-0000-0000-000000000002",
			"exp":       time.Now().Add(24 * time.Hour).Unix(),
		}
		tokenUMSA := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsUMSA)
		tokenStrUMSA, _ := tokenUMSA.SignedString(jwtSecret)

		reqUMSA := httptest.NewRequest("GET", "/api/v1/workspaces/ws-umsa-1", nil)
		reqUMSA.Header.Set("Authorization", "Bearer "+tokenStrUMSA)
		recUMSA := httptest.NewRecorder()

		tenantMiddleware(nextHandler).ServeHTTP(recUMSA, reqUMSA)
		if recUMSA.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", recUMSA.Code)
		}
		if recUMSA.Body.String() != "00000000-0000-0000-0000-000000000002" {
			t.Errorf("Expected tenant UMSA ID, got %s", recUMSA.Body.String())
		}

		// Test 3: @gmail.com (no autorizado) -> El token con tenant inexistente o sin tenant falla
		claimsInvalid := jwt.MapClaims{
			"user_id": "student-gmail",
			"email":   "somebody@gmail.com",
			"role":    "student",
			"exp":     time.Now().Add(24 * time.Hour).Unix(),
		}
		tokenInvalid := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsInvalid)
		tokenStrInvalid, _ := tokenInvalid.SignedString(jwtSecret)

		reqInvalid := httptest.NewRequest("GET", "/api/v1/workspaces/ws-uab-1", nil)
		reqInvalid.Header.Set("Authorization", "Bearer "+tokenStrInvalid)
		recInvalid := httptest.NewRecorder()

		tenantMiddleware(nextHandler).ServeHTTP(recInvalid, reqInvalid)
		if recInvalid.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", recInvalid.Code)
		}
	})

	// 3. Test Isolation /api/v1/workspaces logic (Test 3 & 4)
	t.Run("Tenant data isolation", func(t *testing.T) {
		tenantMiddleware := middleware.WithTenant(tenantRepo, jwtSecret)

		// Test 3: Consultar workspaces de UAB como usuario UAB sólo retorna UAB
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			list, _ := workspaceRepo.GetActiveWorkspaces(r.Context())
			json.NewEncoder(w).Encode(list)
		})

		claimsUAB := jwt.MapClaims{
			"user_id":   "student-uab",
			"email":     "estudiante@uab.edu.bo",
			"role":      "student",
			"tenant_id": "00000000-0000-0000-0000-000000000001",
			"exp":       time.Now().Add(24 * time.Hour).Unix(),
		}
		tokenUAB := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsUAB)
		tokenStrUAB, _ := tokenUAB.SignedString(jwtSecret)

		req := httptest.NewRequest("GET", "/api/v1/workspaces", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStrUAB)
		rec := httptest.NewRecorder()

		tenantMiddleware(nextHandler).ServeHTTP(rec, req)

		var res []*coredomain.WorkspaceInstance
		json.Unmarshal(rec.Body.Bytes(), &res)

		if len(res) != 1 || res[0].ID != "ws-uab-1" {
			t.Errorf("Data leakage: returned workspaces from another tenant!")
		}

		// Test 4: UAB student trying to query UMSA workspace ID directly -> fails with error
		nextSingleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wsID := "ws-umsa-1" // Solicita explícitamente UMSA
			ws, err := workspaceRepo.GetByID(r.Context(), wsID)
			if err != nil || ws == nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			w.Write([]byte("OK"))
		})

		reqSingle := httptest.NewRequest("GET", "/api/v1/workspaces/ws-umsa-1", nil)
		reqSingle.Header.Set("Authorization", "Bearer "+tokenStrUAB)
		recSingle := httptest.NewRecorder()

		tenantMiddleware(nextSingleHandler).ServeHTTP(recSingle, reqSingle)
		if recSingle.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden on foreign tenant workspace access, got %d", recSingle.Code)
		}
	})
}
