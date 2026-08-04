package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	coredomain "solv-backend/internal/core/domain"
	"solv-backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	ErrJWTExpired = errors.New("jwt token has expired")
	ErrJWTInvalid = errors.New("jwt token signature or claims invalid")
)

type AuthService struct {
	repo        domain.UserRepository
	tenantRepo  coredomain.TenantRepository
	oauthConfig *oauth2.Config
	jwtSecret   []byte
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
		repo:        repo,
		tenantRepo:  tenantRepo,
		oauthConfig: config,
		jwtSecret:   []byte(jwtSecret),
	}
}

func (s *AuthService) GetLoginURL() string {
	return s.oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func (s *AuthService) CallbackGoogle(ctx context.Context, code string) (string, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token: %w", err)
	}

	client := s.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

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

	// 1. Obtener todos los tenants para validar el email de forma dinámica
	tenants, err := s.tenantRepo.GetAll(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tenants: %w", err)
	}

	var matchedTenant *coredomain.Tenant
	for _, tenant := range tenants {
		var domains []string
		if len(tenant.AllowedDomains) > 0 {
			if err := json.Unmarshal(tenant.AllowedDomains, &domains); err == nil {
				for _, dom := range domains {
					if strings.HasSuffix(googleUser.Email, dom) {
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
		return "", errors.New("unauthorized: email domain not allowed by any registered tenant")
	}

	user, err := s.repo.GetUserByEmail(ctx, googleUser.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			// Create user
			newUser := &domain.User{
				FirstName: googleUser.GivenName,
				LastName:  googleUser.FamilyName,
				Email:     googleUser.Email,
				Role:      "student", // Default role
				TenantID:  matchedTenant.ID,
			}
			id, createErr := s.repo.CreateUserFromSSO(ctx, newUser)
			if createErr != nil {
				return "", fmt.Errorf("failed to create user: %w", createErr)
			}
			newUser.ID = id
			user = newUser
		} else {
			return "", fmt.Errorf("failed to fetch user: %w", err)
		}
	}

	// Generate JWT token including tenant_id claim
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   user.ID,
		"email":     user.Email,
		"role":      user.Role,
		"tenant_id": user.TenantID,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := jwtToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) ValidateSessionToken(tokenString string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, ErrJWTInvalid
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(err.Error(), "expired") {
			return nil, ErrJWTExpired
		}
		return nil, ErrJWTInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrJWTInvalid
	}

	return claims, nil
}
