package main

import (
	"log"
	"net/http"
	"os"

	"remi-trip-planner/internal/httpapp"
	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func main() {
	dbPath := envOrDefault("SQLITE_PATH", "./data/trips.db")
	addr := envOrDefault("APP_ADDR", ":8080")

	db, err := sqlite.OpenAndMigrate(dbPath, "migrations/001_init.sql")
	if err != nil {
		log.Fatalf("failed to initialize db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewRepository(db)
	service := trips.NewService(repo)

	router := httpapp.NewRouter(httpapp.Dependencies{
		TripService: service,
	})

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
