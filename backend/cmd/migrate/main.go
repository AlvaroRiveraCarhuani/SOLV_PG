package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	_ "github.com/lib/pq"
	"solv-backend/internal/infrastructure/database"
)

func main() {
	var dsn, dir, action, dbFlag string
	flag.StringVar(&dsn, "dsn", "", "database DSN")
	flag.StringVar(&dbFlag, "db", "", "database DSN (alias for -dsn)")
	flag.StringVar(&dir, "dir", "migrations", "migrations directory")
	flag.StringVar(&action, "action", "up", "action to perform: up, down, reset")
	flag.Parse()

	if dsn == "" && dbFlag != "" {
		dsn = dbFlag
	}
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = os.Getenv("TEST_DB_DSN")
	}
	if dsn == "" {
		dsn = "postgres://solv_user:solv_password@127.0.0.1:5432/solv_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Fatal: failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Fatal: failed to ping database: %v", err)
	}

	switch action {
	case "down":
		log.Printf("Reverting all migrations from %s ...", dir)
		if err := database.RunMigrationsDown(db, dir); err != nil {
			log.Fatalf("Fatal: migration down failed: %v", err)
		}
		log.Println("Migrations reverted successfully.")

	case "reset":
		log.Printf("Resetting (down then up) migrations from %s ...", dir)
		if err := database.RunMigrationsDown(db, dir); err != nil {
			log.Fatalf("Fatal: migration down failed: %v", err)
		}
		if err := database.RunMigrations(db, dir); err != nil {
			log.Fatalf("Fatal: migration up failed: %v", err)
		}
		log.Println("Migration reset completed successfully.")

	default:
		log.Printf("Applying migrations from %s ...", dir)
		if err := database.RunMigrations(db, dir); err != nil {
			log.Fatalf("Fatal: migration failed: %v", err)
		}
		log.Println("Migrations applied successfully.")
	}
}
