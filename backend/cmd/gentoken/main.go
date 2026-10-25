package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

func main() {
	role := flag.String("role", "admin", "User role (admin, teacher, student)")
	userID := flag.String("user_id", "", "User UUID")
	tenantID := flag.String("tenant_id", "00000000-0000-0000-0000-000000000001", "Tenant UUID")
	email := flag.String("email", "", "User email")
	days := flag.Int("days", 30, "Token validity in days")
	secret := flag.String("secret", "", "JWT secret key")
	rawOnly := flag.Bool("raw", false, "Print only the raw token string")
	flag.Parse()

	jwtSecret := *secret
	if jwtSecret == "" {
		jwtSecret = os.Getenv("JWT_SECRET")
	}
	jwtSecret = strings.Trim(strings.TrimSpace(jwtSecret), `"`)
	if jwtSecret == "" {
		jwtSecret = "una_cadena_aleatoria_y_muy_larga_para_solv"
	}

	userEmail := strings.TrimSpace(*email)
	currentUserID := strings.TrimSpace(*userID)
	currentTenantID := strings.TrimSpace(*tenantID)
	currentRole := strings.TrimSpace(*role)

	// Resolver usuario real desde PostgreSQL si DATABASE_URL esta presente
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		if db, err := sql.Open("postgres", dbURL); err == nil {
			defer db.Close()

			var qID, qEmail, qRole, qTenantID string
			var query string
			var args []interface{}

			if currentUserID != "" {
				query = "SELECT id, email, role, tenant_id FROM users WHERE id = $1 LIMIT 1"
				args = []interface{}{currentUserID}
			} else if userEmail != "" {
				query = "SELECT id, email, role, tenant_id FROM users WHERE LOWER(email) = LOWER($1) LIMIT 1"
				args = []interface{}{userEmail}
			} else {
				query = "SELECT id, email, role, tenant_id FROM users WHERE role = $1 AND status = 'active' ORDER BY created_at ASC LIMIT 1"
				args = []interface{}{currentRole}
			}

			err = db.QueryRow(query, args...).Scan(&qID, &qEmail, &qRole, &qTenantID)
			if err == nil {
				currentUserID = qID
				userEmail = qEmail
				currentRole = qRole
				if qTenantID != "" {
					currentTenantID = qTenantID
				}
				if !*rawOnly {
					fmt.Fprintf(os.Stderr, "Usuario resuelto desde BD: %s (%s) [Rol: %s]\n", userEmail, currentUserID, currentRole)
				}
			} else if currentUserID == "" {
				// No se encontro: aprovisionar usuario dev en la BD para que /api/v1/users/me nunca retorne 404
				if userEmail == "" {
					switch currentRole {
					case "teacher":
						userEmail = "docente.dev@uab.edu.bo"
					case "student":
						userEmail = "estudiante.dev@uab.edu.bo"
					default:
						userEmail = "admin.dev@uab.edu.bo"
					}
				}

				firstName := strings.ToUpper(currentRole[:1]) + currentRole[1:]
				lastName := "Dev"
				insQuery := `
					INSERT INTO users (id, first_name, last_name, email, role, tenant_id, status)
					VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'active')
					ON CONFLICT (email) DO UPDATE SET role = EXCLUDED.role
					RETURNING id, email, role, tenant_id`

				err = db.QueryRow(insQuery, firstName, lastName, userEmail, currentRole, currentTenantID).Scan(&qID, &qEmail, &qRole, &qTenantID)
				if err == nil {
					currentUserID = qID
					userEmail = qEmail
					currentRole = qRole
					if qTenantID != "" {
						currentTenantID = qTenantID
					}
					if !*rawOnly {
						fmt.Fprintf(os.Stderr, "Usuario semilla aprovisionado en BD: %s (%s) [Rol: %s]\n", userEmail, currentUserID, currentRole)
					}
				}
			}
		}
	}

	// Fallback si no hay BD o fallo la conexion
	if currentUserID == "" {
		currentUserID = "00000000-0000-0000-0000-000000000001"
	}
	if userEmail == "" {
		switch currentRole {
		case "teacher":
			userEmail = "docente@solv.edu.bo"
		case "student":
			userEmail = "estudiante@solv.edu.bo"
		default:
			userEmail = "admin@solv.edu.bo"
		}
	}

	claims := jwt.MapClaims{
		"user_id":   currentUserID,
		"email":     userEmail,
		"role":      currentRole,
		"tenant_id": currentTenantID,
		"exp":       time.Now().Add(time.Duration(*days) * 24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing token: %v\n", err)
		os.Exit(1)
	}

	if *rawOnly {
		fmt.Print(tokenStr)
		return
	}

	fmt.Println(tokenStr)
}
