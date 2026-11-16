// Comando de operación para el alta de instituciones.
//
// Una institución nueva no se crea desde el panel porque el panel pertenece a
// una institución: quien crea otra no tiene sesión en ninguna. Este comando
// corre en el host (o dentro del contenedor del backend) y hace el alta sin SQL.
//
//	docker compose exec backend solv-tenant -action create \
//	  -name "Universidad Ejemplo" -slug uejemplo \
//	  -admin-email admin@uejemplo.edu -domains uejemplo.edu \
//	  -period-code 2026-I -period-name "Primer Semestre 2026" \
//	  -period-start 2026-02-01 -period-end 2026-06-30
//
//	docker compose exec backend solv-tenant -action list
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func main() {
	var (
		dsn, action, name, slug, adminEmail, adminFirst, adminLast, domains string
		periodCode, periodName, periodStart, periodEnd                      string
	)

	flag.StringVar(&dsn, "dsn", "", "database DSN (por defecto DATABASE_URL)")
	flag.StringVar(&action, "action", "create", "acción: create, list")
	flag.StringVar(&name, "name", "", "nombre de la institución")
	flag.StringVar(&slug, "slug", "", "identificador corto (minúsculas, números y guiones)")
	flag.StringVar(&adminEmail, "admin-email", "", "correo del primer administrador")
	flag.StringVar(&adminFirst, "admin-first-name", "Administrador", "nombre del primer administrador")
	flag.StringVar(&adminLast, "admin-last-name", "Institucional", "apellido del primer administrador")
	flag.StringVar(&domains, "domains", "", "dominios permitidos, separados por coma (ej: uejemplo.edu,uejemplo.edu.bo)")
	flag.StringVar(&periodCode, "period-code", "", "código del periodo (por defecto AAAA-I)")
	flag.StringVar(&periodName, "period-name", "", "nombre del periodo")
	flag.StringVar(&periodStart, "period-start", "", "inicio del periodo (AAAA-MM-DD)")
	flag.StringVar(&periodEnd, "period-end", "", "fin del periodo (AAAA-MM-DD)")
	flag.Parse()

	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		log.Fatal("falta -dsn o DATABASE_URL")
	}

	ctx := context.Background()
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base: %v", err)
	}
	defer func() { _ = db.GetDB().Close() }()

	repo := postgres.NewPostgresTenantProvisioningRepository(db.GetDB())

	switch action {
	case "list":
		tenants, err := repo.ListTenants(ctx)
		if err != nil {
			log.Fatalf("error listando instituciones: %v", err)
		}
		if len(tenants) == 0 {
			fmt.Println("No hay instituciones en esta instalación.")
			return
		}
		fmt.Printf("%-36s  %-20s  %-40s  %s\n", "ID", "IDENTIFICADOR", "NOMBRE", "CREADA")
		for _, t := range tenants {
			fmt.Printf("%-36s  %-20s  %-40s  %s\n", t.ID, t.Slug, t.Name, t.CreatedAt.Format("2006-01-02"))
		}

	case "create":
		start, err := parseDate(periodStart, defaultPeriodStart())
		if err != nil {
			log.Fatalf("fecha de inicio inválida: %v", err)
		}
		end, err := parseDate(periodEnd, defaultPeriodEnd())
		if err != nil {
			log.Fatalf("fecha de fin inválida: %v", err)
		}

		req := domain.TenantProvisioningRequest{
			Name:           name,
			Slug:           slug,
			AllowedDomains: splitDomains(domains),
			AdminEmail:     adminEmail,
			AdminFirstName: adminFirst,
			AdminLastName:  adminLast,
			PeriodCode:     periodCode,
			PeriodName:     periodName,
			PeriodStart:    start,
			PeriodEnd:      end,
		}

		svc := services.NewTenantProvisioningService(repo)
		result, err := svc.Provision(ctx, req)
		if errors.Is(err, domain.ErrTenantSlugExists) {
			// Idempotencia para el instalador: reejecutar no falla.
			fmt.Printf("La institución %q ya existe: sin cambios.\n", req.Slug)
			return
		}
		if err != nil {
			log.Fatalf("no se pudo crear la institución: %v", err)
		}

		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		fmt.Printf("Institución %q creada. Primer administrador: %s\n", req.Name, req.AdminEmail)

	default:
		log.Fatalf("acción desconocida %q (usá create o list)", action)
	}
}

func splitDomains(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func parseDate(raw string, fallback time.Time) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	return time.Parse("2006-01-02", strings.TrimSpace(raw))
}

// El periodo por defecto es el año calendario en curso: quien no pasa fechas
// explícitas igual queda con un periodo activo para crear materias.
func defaultPeriodStart() time.Time {
	y := time.Now().Year()
	return time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC)
}

func defaultPeriodEnd() time.Time {
	y := time.Now().Year()
	return time.Date(y, time.December, 31, 0, 0, 0, 0, time.UTC)
}
