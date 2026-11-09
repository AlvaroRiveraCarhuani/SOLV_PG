package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	coredomain "solv-backend/internal/core/domain"
	"solv-backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	ErrJWTExpired                   = errors.New("jwt token has expired")
	ErrJWTInvalid                   = errors.New("jwt token signature or claims invalid")
	ErrJWTRevoked                   = errors.New("jwt session has been revoked")
	ErrSessionRevocationUnavailable = errors.New("session revocation store unavailable")
	ErrUnauthorizedDomain           = errors.New("unauthorized: email domain not allowed by any registered tenant")
	ErrEmailNotVerified             = errors.New("unauthorized: google email is not verified")
	ErrOAuthStateMissing            = errors.New("oauth state is missing")
	ErrOAuthStateInvalid            = errors.New("oauth state is invalid or already used")
	ErrOAuthStateExpired            = errors.New("oauth state has expired")
)

const SessionRevocationPollInterval = 2 * time.Second
const oauthStateTTL = 10 * time.Minute

type AuthService struct {
	repo            domain.UserRepository
	tenantRepo      coredomain.TenantRepository
	teacherInvRepo  coredomain.TeacherInvitationRepository
	oauthConfig     *oauth2.Config
	jwtSecret       []byte
	revocationStore coredomain.SessionRevocationRepository
	oauthStates     *oauthStateStore
	sessionMu       sync.Mutex
	listeners       map[uint64]func(string)
	tokenListeners  map[string]map[uint64]func()
	listenerSeq     uint64
}

type oauthStateStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
	ttl     time.Duration
	now     func() time.Time
}

func newOAuthStateStore(ttl time.Duration) *oauthStateStore {
	return &oauthStateStore{
		entries: make(map[string]time.Time),
		ttl:     ttl,
		now:     time.Now,
	}
}

func (s *oauthStateStore) issue() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(raw)

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for candidate, expiresAt := range s.entries {
		if !expiresAt.After(now) {
			delete(s.entries, candidate)
		}
	}
	s.entries[state] = now.Add(s.ttl)
	return state, nil
}

func (s *oauthStateStore) consume(state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiresAt, ok := s.entries[state]
	if !ok {
		return ErrOAuthStateInvalid
	}
	delete(s.entries, state)
	if !expiresAt.After(s.now()) {
		return ErrOAuthStateExpired
	}
	return nil
}

type googleIdentity struct {
	Email         string
	GivenName     string
	FamilyName    string
	VerifiedEmail bool
}

func NewAuthService(repo domain.UserRepository, tenantRepo coredomain.TenantRepository) *AuthService {
	cleanEnv := func(key string) string {
		return strings.Trim(strings.TrimSpace(os.Getenv(key)), `"`)
	}

	clientID := cleanEnv("GOOGLE_CLIENT_ID")
	clientSecret := cleanEnv("GOOGLE_CLIENT_SECRET")
	redirectURL := cleanEnv("GOOGLE_REDIRECT_URL")
	jwtSecret := cleanEnv("JWT_SECRET")

	if clientID == "" || clientSecret == "" || redirectURL == "" || jwtSecret == "" {
		panic("Auth service requires GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL and JWT_SECRET")
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthService{
		repo:           repo,
		tenantRepo:     tenantRepo,
		oauthConfig:    config,
		jwtSecret:      []byte(jwtSecret),
		oauthStates:    newOAuthStateStore(oauthStateTTL),
		listeners:      make(map[uint64]func(string)),
		tokenListeners: make(map[string]map[uint64]func()),
	}
}

func (s *AuthService) WithTeacherInvitationRepo(repo coredomain.TeacherInvitationRepository) *AuthService {
	s.teacherInvRepo = repo
	return s
}

func (s *AuthService) WithSessionRevocationRepository(repo coredomain.SessionRevocationRepository) *AuthService {
	s.revocationStore = repo
	return s
}

// BeginGoogleLogin genera un state de un solo uso y devuelve la URL de Google
// junto con el valor que el handler debe asociar a la cookie del navegador.
func (s *AuthService) BeginGoogleLogin() (string, string, error) {
	state, err := s.oauthStates.issue()
	if err != nil {
		return "", "", err
	}
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline), state, nil
}

// ConsumeOAuthState valida y consume un state. Un state reutilizado o
// desconocido devuelve ErrOAuthStateInvalid; uno vencido devuelve
// ErrOAuthStateExpired. La comparación con la cookie la hace el handler con
// comparación de tiempo constante.
func (s *AuthService) ConsumeOAuthState(state string) error {
	state = strings.TrimSpace(state)
	if state == "" {
		return ErrOAuthStateMissing
	}
	return s.oauthStates.consume(state)
}

func (s *AuthService) CallbackGoogle(ctx context.Context, code string) (string, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}

	client := s.oauthConfig.Client(ctx, token)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var googleUser struct {
		Email         string `json:"email"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	if err := json.Unmarshal(data, &googleUser); err != nil {
		return "", fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	user, err := s.provisionOAuthUser(ctx, googleIdentity{
		Email:         googleUser.Email,
		GivenName:     googleUser.GivenName,
		FamilyName:    googleUser.FamilyName,
		VerifiedEmail: googleUser.VerifiedEmail,
	})
	if err != nil {
		return "", err
	}
	return s.issueSessionToken(user)
}

func emailMatchesDomain(email, domain string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "@")
	if domain == "" {
		return false
	}
	at := strings.LastIndex(email, "@")
	return at > 0 && at < len(email)-1 && email[at+1:] == domain
}

// provisionOAuthUser valida la identidad de Google y resuelve el usuario y su
// tenant. Es la parte del callback que no depende de la red, para poder
// probarla sin credenciales reales.
func (s *AuthService) provisionOAuthUser(ctx context.Context, googleUser googleIdentity) (*domain.User, error) {
	if !googleUser.VerifiedEmail {
		return nil, ErrEmailNotVerified
	}

	if strings.TrimSpace(googleUser.Email) == "" {
		return nil, ErrUnauthorizedDomain
	}

	// 1. Obtener todos los tenants para validar el email de forma dinámica
	tenants, err := s.tenantRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenants: %w", err)
	}

	var matchedTenant *coredomain.Tenant
	for _, tenant := range tenants {
		var domains []string
		if len(tenant.AllowedDomains) > 0 {
			if err := json.Unmarshal(tenant.AllowedDomains, &domains); err == nil {
				for _, dom := range domains {
					if emailMatchesDomain(googleUser.Email, dom) {
						matchedTenant = tenant
						break
					}
				}
			}
		}
		if matchedTenant != nil {
			break
		}
	}

	if matchedTenant == nil {
		return nil, ErrUnauthorizedDomain
	}

	// Verificar si el correo tiene una invitación docente activa y consumirla
	hasTeacherInvite := false
	if s.teacherInvRepo != nil {
		if invited, err := s.teacherInvRepo.CheckAndConsumeInvitation(ctx, matchedTenant.ID, googleUser.Email); err == nil && invited {
			hasTeacherInvite = true
		}
	}

	user, err := s.repo.GetUserByEmail(ctx, googleUser.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			targetRole := "student"
			if hasTeacherInvite {
				targetRole = "teacher"
			}
			// Create user
			newUser := &domain.User{
				FirstName: googleUser.GivenName,
				LastName:  googleUser.FamilyName,
				Email:     googleUser.Email,
				Role:      targetRole,
				TenantID:  matchedTenant.ID,
			}
			id, createErr := s.repo.CreateUserFromSSO(ctx, newUser)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create user: %w", createErr)
			}
			newUser.ID = id
			user = newUser
		} else {
			return nil, fmt.Errorf("failed to fetch user: %w", err)
		}
	} else if hasTeacherInvite && user.Role == "student" {
		// Promover a docente si era estudiante previo pero fue formalmente invitado como docente
		if err := s.repo.UpdateUserRole(ctx, user.ID, "teacher"); err == nil {
			user.Role = "teacher"
		}
	}

	return user, nil
}

func (s *AuthService) issueSessionToken(user *domain.User) (string, error) {
	jwtTTL := 24 * time.Hour
	if ttlStr := strings.TrimSpace(os.Getenv("JWT_TTL_HOURS")); ttlStr != "" {
		if hours, err := strconv.Atoi(ttlStr); err == nil && hours > 0 {
			if hours > 24 {
				hours = 24
			}
			jwtTTL = time.Duration(hours) * time.Hour
		}
	}

	// Generate JWT token including tenant_id claim
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   user.ID,
		"email":     user.Email,
		"role":      user.Role,
		"tenant_id": user.TenantID,
		"jti":       uuid.NewString(),
		"exp":       time.Now().Add(jwtTTL).Unix(),
	})

	tokenString, err := jwtToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateSessionToken conserva la firma antigua para consumidores que no
// disponen del contexto de la petición. La autenticación HTTP y WebSocket debe
// usar ValidateSessionTokenContext.
func (s *AuthService) ValidateSessionToken(tokenString string) (jwt.MapClaims, error) {
	return s.ValidateSessionTokenContext(context.Background(), tokenString)
}

func (s *AuthService) ValidateSessionTokenContext(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	claims, err := s.parseSessionClaims(tokenString, true)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSessionNotRevoked(ctx, tokenString, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (s *AuthService) parseSessionClaims(tokenString string, validateClaims bool) (jwt.MapClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrJWTInvalid
	}

	options := make([]jwt.ParserOption, 0, 1)
	if !validateClaims {
		options = append(options, jwt.WithoutClaimsValidation())
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	}, options...)

	if err != nil {
		if validateClaims && (errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(err.Error(), "expired")) {
			return nil, ErrJWTExpired
		}
		return nil, ErrJWTInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || (validateClaims && !token.Valid) {
		return nil, ErrJWTInvalid
	}
	return claims, nil
}

func sessionTokenHash(tokenString string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(tokenString)))
	return hex.EncodeToString(sum[:])
}

// SessionIdentifierFromClaims devuelve el identificador persistido de la
// sesión. Los tokens nuevos usan su jti; los heredados sin jti usan un hash
// SHA-256 para no guardar nunca el JWT completo.
func SessionIdentifierFromClaims(tokenString string, claims jwt.MapClaims) string {
	if jti, ok := claims["jti"].(string); ok {
		if jti = strings.TrimSpace(jti); jti != "" {
			return "jti:" + jti
		}
	}
	return "legacy:" + sessionTokenHash(tokenString)
}

func (s *AuthService) ensureSessionNotRevoked(ctx context.Context, tokenString string, claims jwt.MapClaims) error {
	if s.revocationStore == nil {
		return ErrSessionRevocationUnavailable
	}
	sessionID := SessionIdentifierFromClaims(tokenString, claims)
	revoked, err := s.revocationStore.IsSessionRevoked(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSessionRevocationUnavailable, err)
	}
	if revoked {
		return ErrJWTRevoked
	}
	return nil
}

// RevokeSessionToken persiste la revocación antes de notificar a las
// conexiones locales. Si el repositorio falla, el logout debe fallar y no debe
// presentarse como exitoso.
func (s *AuthService) RevokeSessionToken(ctx context.Context, tokenString string) error {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil
	}

	claims, err := s.parseSessionClaims(tokenString, false)
	if err != nil {
		// Un token inválido no representa una sesión que pueda revocarse; el
		// handler igualmente puede limpiar la cookie.
		return nil
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil || !exp.Time.After(time.Now()) {
		return nil
	}

	if s.revocationStore == nil {
		return ErrSessionRevocationUnavailable
	}
	sessionID := SessionIdentifierFromClaims(tokenString, claims)
	if err := s.revocationStore.RevokeSession(ctx, sessionID, exp.Time); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionRevocationUnavailable, err)
	}

	s.sessionMu.Lock()
	global := make([]func(string), 0, len(s.listeners))
	for _, listener := range s.listeners {
		global = append(global, listener)
	}
	perToken := make([]func(), 0)
	for _, listener := range s.tokenListeners[sessionID] {
		perToken = append(perToken, listener)
	}
	s.sessionMu.Unlock()

	for _, listener := range global {
		listener(sessionID)
	}
	for _, listener := range perToken {
		listener()
	}
	return nil
}

// IsSessionRevoked conserva la firma booleana y falla cerrado: un error del
// repositorio se interpreta como sesión no verificable.
func (s *AuthService) IsSessionRevoked(tokenString string) bool {
	revoked, err := s.IsSessionRevokedContext(context.Background(), tokenString)
	return err != nil || revoked
}

func (s *AuthService) IsSessionRevokedContext(ctx context.Context, tokenString string) (bool, error) {
	claims, err := s.parseSessionClaims(tokenString, false)
	if err != nil {
		return false, nil
	}
	return s.IsSessionIDRevokedContext(ctx, SessionIdentifierFromClaims(tokenString, claims))
}

func (s *AuthService) IsSessionIDRevokedContext(ctx context.Context, sessionID string) (bool, error) {
	if s.revocationStore == nil {
		return false, ErrSessionRevocationUnavailable
	}
	revoked, err := s.revocationStore.IsSessionRevoked(ctx, sessionID)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrSessionRevocationUnavailable, err)
	}
	return revoked, nil
}

// IsSessionHashRevoked conserva la comprobación booleana anterior; el valor
// recibido es ahora el identificador de sesión completo.
func (s *AuthService) IsSessionHashRevoked(sessionID string) bool {
	revoked, err := s.IsSessionIDRevokedContext(context.Background(), sessionID)
	return err != nil || revoked
}

// WatchSessionRevocation cierra una conexión si otra instancia revoca su
// sesión. La comprobación periódica es el mecanismo de propagación entre
// procesos; una operación posterior de autorización también es rechazada por
// la consulta persistente.
func (s *AuthService) WatchSessionRevocation(ctx context.Context, sessionID string, onRevoked func()) {
	if onRevoked == nil {
		return
	}
	ticker := time.NewTicker(SessionRevocationPollInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				revoked, err := s.IsSessionIDRevokedContext(ctx, sessionID)
				if err != nil || revoked {
					onRevoked()
					return
				}
			}
		}
	}()
}

// RegisterRevocationListener recibe el hash de cada sesión revocada. Se usa
// para cerrar conexiones WebSocket abiertas sin acoplar el servicio de auth al
// hub de delivery.
func (s *AuthService) RegisterRevocationListener(listener func(string)) func() {
	s.sessionMu.Lock()
	if s.listeners == nil {
		s.listeners = make(map[uint64]func(string))
	}
	s.listenerSeq++
	id := s.listenerSeq
	s.listeners[id] = listener
	s.sessionMu.Unlock()

	return func() {
		s.sessionMu.Lock()
		delete(s.listeners, id)
		s.sessionMu.Unlock()
	}
}

// RegisterTokenRevocationListener permite que una conexión concreta cierre su
// socket cuando se revoque exactamente su token.
func (s *AuthService) RegisterTokenRevocationListener(hash string, listener func()) func() {
	s.sessionMu.Lock()
	if s.tokenListeners == nil {
		s.tokenListeners = make(map[string]map[uint64]func())
	}
	if s.tokenListeners[hash] == nil {
		s.tokenListeners[hash] = make(map[uint64]func())
	}
	s.listenerSeq++
	id := s.listenerSeq
	s.tokenListeners[hash][id] = listener
	s.sessionMu.Unlock()

	return func() {
		s.sessionMu.Lock()
		delete(s.tokenListeners[hash], id)
		if len(s.tokenListeners[hash]) == 0 {
			delete(s.tokenListeners, hash)
		}
		s.sessionMu.Unlock()
	}
}
