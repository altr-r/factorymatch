package main

import (
	"log"
	"net/http"

	"factorymatch/internal/config"
	"factorymatch/internal/db"
	"factorymatch/internal/handlers"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if database != nil {
		if err := db.Migrate(database); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	} else {
		log.Println("DATABASE_URL not set - running without a database (DB routes return 503)")
	}

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, database, cfg)
	log.Printf("listening on: %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
