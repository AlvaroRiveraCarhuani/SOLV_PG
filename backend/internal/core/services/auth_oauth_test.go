package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	coredomain "solv-backend/internal/core/domain"
	"solv-backend/internal/domain"
)

type oauthTenantRepo struct {
	tenants []*coredomain.Tenant
}

func (m *oauthTenantRepo) GetByID(_ context.Context, id string) (*coredomain.Tenant, error) {
	for _, tenant := range m.tenants {
		if tenant.ID == id {
			return tenant, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *oauthTenantRepo) GetBySlug(_ context.Context, slug string) (*coredomain.Tenant, error) {
	for _, tenant := range m.tenants {
		if tenant.Slug == slug {
			return tenant, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *oauthTenantRepo) GetAll(context.Context) ([]*coredomain.Tenant, error) {
	return m.tenants, nil
}

func (m *oauthTenantRepo) UpdateConfig(context.Context, string, []byte) error { return nil }
func (m *oauthTenantRepo) SetMaintenance(context.Context, string, bool, *time.Time, string) error {
	return nil
}
func (m *oauthTenantRepo) GetMaintenance(context.Context, string) (*coredomain.MaintenanceStatus, error) {
	return &coredomain.MaintenanceStatus{}, nil
}

type oauthUserRepo struct {
	users  map[string]*domain.User
	nextID int
}

func (m *oauthUserRepo) CreateUser(context.Context, domain.CreateUserDTO) (string, error) {
	return "", errors.New("not used")
}

func (m *oauthUserRepo) GetUserByID(context.Context, string) (domain.UserResponseDTO, error) {
	return domain.UserResponseDTO{}, errors.New("not used")
}

func (m *oauthUserRepo) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range m.users {
		if strings.EqualFold(user.Email, email) {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *oauthUserRepo) CreateUserFromSSO(_ context.Context, user *domain.User) (string, error) {
	m.nextID++
	id := fmt.Sprintf("user-%d", m.nextID)
	copy := *user
	copy.ID = id
	m.users[id] = &copy
	return id, nil
}

func (m *oauthUserRepo) UpdateUserRole(_ context.Context, id, role string) error {
	user, ok := m.users[id]
	if !ok {
		return errors.New("user not found")
	}
	user.Role = role
	return nil
}

type oauthInvitationRepo struct {
	coredomain.TeacherInvitationRepository
	invited bool
}

func (m *oauthInvitationRepo) CheckAndConsumeInvitation(context.Context, string, string) (bool, error) {
	return m.invited, nil
}

func newOAuthTestAuth(t *testing.T, userRepo domain.UserRepository, tenantRepo coredomain.TenantRepository) *AuthService {
	t.Helper()
	t.Setenv("GOOGLE_CLIENT_ID", "oauth-test-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "oauth-test-secret")
	t.Setenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/v1/auth/callback")
	t.Setenv("JWT_SECRET", "oauth-test-jwt-secret")
	return NewAuthService(userRepo, tenantRepo)
}

func oauthTestTenants() *oauthTenantRepo {
	return &oauthTenantRepo{tenants: []*coredomain.Tenant{
		{ID: "tenant-uab", Name: "UAB", Slug: "uab", AllowedDomains: []byte(`["@uab.edu.bo"]`)},
		{ID: "tenant-umsa", Name: "UMSA", Slug: "umsa", AllowedDomains: []byte(`["@umsa.edu.bo"]`)},
	}}
}

func TestOAuthProvisioningCreatesStudentAndIssuesRequiredClaims(t *testing.T) {
	users := &oauthUserRepo{users: map[string]*domain.User{}}
	authService := newOAuthTestAuth(t, users, oauthTestTenants())

	user, err := authService.provisionOAuthUser(context.Background(), googleIdentity{
		Email:         "ada@uab.edu.bo",
		GivenName:     "Ada",
		FamilyName:    "Lovelace",
		VerifiedEmail: true,
	})
	if err != nil {
		t.Fatalf("el aprovisionamiento OAuth falló: %v", err)
	}
	if user.ID == "" || user.TenantID != "tenant-uab" || user.Role != "student" {
		t.Fatalf("usuario OAuth inesperado: %+v", user)
	}

	token, err := authService.issueSessionToken(user)
	if err != nil {
		t.Fatalf("no se pudo emitir el JWT: %v", err)
	}
	claims, err := authService.parseSessionClaims(token, true)
	if err != nil {
		t.Fatalf("el JWT emitido no es válido: %v", err)
	}
	for _, key := range []string{"user_id", "email", "role", "tenant_id", "jti"} {
		if value, _ := claims[key].(string); strings.TrimSpace(value) == "" {
			t.Fatalf("el claim %q está vacío: %+v", key, claims)
		}
	}
	if claims["user_id"] != user.ID || claims["tenant_id"] != "tenant-uab" || claims["role"] != "student" {
		t.Fatalf("los claims no coinciden con el usuario aprovisionado: %+v", claims)
	}
	exp, ok := claims["exp"].(float64)
	if !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
		t.Fatalf("el JWT debe incluir una expiración futura: %+v", claims)
	}
}

func TestOAuthProvisioningRejectsUnverifiedAndForeignDomains(t *testing.T) {
	authService := newOAuthTestAuth(t, &oauthUserRepo{users: map[string]*domain.User{}}, oauthTestTenants())

	if _, err := authService.provisionOAuthUser(context.Background(), googleIdentity{
		Email:         "ada@uab.edu.bo",
		VerifiedEmail: false,
	}); !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("un correo no verificado debe rechazarse, se obtuvo %v", err)
	}

	for _, email := range []string{
		"ada@evil-uab.edu.bo",
		"ada@uab.edu.bo.evil.example",
		"ada@umsa.edu.bo.evil.example",
		"ada@evil-umsa.edu.bo",
	} {
		if _, err := authService.provisionOAuthUser(context.Background(), googleIdentity{
			Email:         email,
			VerifiedEmail: true,
		}); !errors.Is(err, ErrUnauthorizedDomain) {
			t.Fatalf("el correo %q debe rechazarse por dominio, se obtuvo %v", email, err)
		}
	}

	user, err := authService.provisionOAuthUser(context.Background(), googleIdentity{
		Email:         "ADA@UAB.EDU.BO",
		GivenName:     "Ada",
		VerifiedEmail: true,
	})
	if err != nil || user.TenantID != "tenant-uab" {
		t.Fatalf("el dominio válido debe compararse sin distinguir mayúsculas: user=%+v err=%v", user, err)
	}

	umsaUser, err := authService.provisionOAuthUser(context.Background(), googleIdentity{
		Email:         "ada@umsa.edu.bo",
		GivenName:     "Ada",
		VerifiedEmail: true,
	})
	if err != nil || umsaUser.TenantID != "tenant-umsa" {
		t.Fatalf("un dominio permitido debe resolver su propio tenant: user=%+v err=%v", umsaUser, err)
	}
}

func TestOAuthProvisioningAppliesTeacherInvitation(t *testing.T) {
	users := &oauthUserRepo{users: map[string]*domain.User{}}
	authService := newOAuthTestAuth(t, users, oauthTestTenants()).
		WithTeacherInvitationRepo(&oauthInvitationRepo{invited: true})

	user, err := authService.provisionOAuthUser(context.Background(), googleIdentity{
		Email:         "docente@uab.edu.bo",
		GivenName:     "Docente",
		VerifiedEmail: true,
	})
	if err != nil {
		t.Fatalf("el aprovisionamiento docente falló: %v", err)
	}
	if user.Role != "teacher" {
		t.Fatalf("una invitación docente debe crear rol teacher, se obtuvo %q", user.Role)
	}
}

func TestOAuthLoginURLUsesConfiguredGoogleClient(t *testing.T) {
	authService := newOAuthTestAuth(t, &oauthUserRepo{users: map[string]*domain.User{}}, oauthTestTenants())

	loginURL, state, err := authService.BeginGoogleLogin()
	if err != nil {
		t.Fatalf("no se pudo iniciar el login: %v", err)
	}
	parsed, err := url.Parse(loginURL)
	if err != nil {
		t.Fatalf("la URL de login no es válida: %v", err)
	}
	if parsed.Host != "accounts.google.com" {
		t.Fatalf("el login debe ir a Google, se obtuvo %q", parsed.Host)
	}
	query := parsed.Query()
	if query.Get("client_id") != "oauth-test-client" {
		t.Fatalf("client_id inesperado: %q", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "http://localhost:3000/api/v1/auth/callback" {
		t.Fatalf("redirect_uri inesperado: %q", query.Get("redirect_uri"))
	}
	if !strings.Contains(query.Get("scope"), "userinfo.email") {
		t.Fatalf("el login debe solicitar el correo: %q", query.Get("scope"))
	}
	if state == "" || query.Get("state") != state {
		t.Fatalf("la URL debe llevar el state emitido: state=%q query=%q", state, query.Get("state"))
	}
}

func TestOAuthStateIsRandomSingleUseAndExpires(t *testing.T) {
	authService := newOAuthTestAuth(t, &oauthUserRepo{users: map[string]*domain.User{}}, oauthTestTenants())

	_, first, err := authService.BeginGoogleLogin()
	if err != nil {
		t.Fatalf("primer login falló: %v", err)
	}
	_, second, err := authService.BeginGoogleLogin()
	if err != nil {
		t.Fatalf("segundo login falló: %v", err)
	}
	if first == "" || second == "" || first == second {
		t.Fatalf("cada login debe generar un state distinto: %q %q", first, second)
	}

	if err := authService.ConsumeOAuthState(first); err != nil {
		t.Fatalf("el primer state debe consumirse una vez: %v", err)
	}
	if err := authService.ConsumeOAuthState(first); !errors.Is(err, ErrOAuthStateInvalid) {
		t.Fatalf("un state reutilizado debe rechazarse, se obtuvo %v", err)
	}
	if err := authService.ConsumeOAuthState(""); !errors.Is(err, ErrOAuthStateMissing) {
		t.Fatalf("un state vacío debe rechazarse, se obtuvo %v", err)
	}

	_, expiring, err := authService.BeginGoogleLogin()
	if err != nil {
		t.Fatalf("no se pudo emitir el state a expirar: %v", err)
	}
	authService.oauthStates.now = func() time.Time { return time.Now().Add(oauthStateTTL + time.Second) }
	if err := authService.ConsumeOAuthState(expiring); !errors.Is(err, ErrOAuthStateExpired) {
		t.Fatalf("un state vencido debe rechazarse, se obtuvo %v", err)
	}
}
