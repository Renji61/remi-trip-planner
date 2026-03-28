package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"remi-trip-planner/internal/httpapp"
	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

// chdirToModuleRoot sets the process cwd to the REMI project root so
// web/templates and web/static resolve correctly. Order: executable dir (when
// the binary lives in the repo), REMI_ROOT env, then walk upward from Getwd()
// for a go.mod next to web/templates.
func chdirToModuleRoot() {
	tryChdir := func(dir string) bool {
		if dir == "" {
			return false
		}
		if _, err := os.Stat(filepath.Join(dir, "web", "templates")); err != nil {
			return false
		}
		if err := os.Chdir(dir); err != nil {
			log.Printf("warning: could not chdir to %s: %v", dir, err)
			return false
		}
		return true
	}

	if env := filepath.Clean(strings.TrimSpace(os.Getenv("REMI_ROOT"))); env != "." && env != "" {
		if tryChdir(env) {
			return
		}
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Clean(filepath.Dir(exe))
		if tryChdir(exeDir) {
			return
		}
		// go build -o bin/remi-server.exe leaves the binary under repo/bin; templates are in the parent dir.
		parent := filepath.Clean(filepath.Join(exeDir, ".."))
		if parent != exeDir && tryChdir(parent) {
			return
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return
	}
	if tryChdir(wd) {
		return
	}
	// Workspace opened at a parent folder (e.g. …/Code Projects); pick this module only.
	if entries, err := os.ReadDir(wd); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			candidate := filepath.Join(wd, e.Name())
			modData, err := os.ReadFile(filepath.Join(candidate, "go.mod"))
			if err != nil || !strings.Contains(string(modData), "module remi-trip-planner") {
				continue
			}
			if tryChdir(candidate) {
				return
			}
		}
	}
	dir := wd
	for {
		if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
			if tryChdir(dir) {
				return
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func main() {
	chdirToModuleRoot()

	dbPath := envOrDefault("SQLITE_PATH", "./data/trips.db")
	addr := envOrDefault("APP_ADDR", ":4122")

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
