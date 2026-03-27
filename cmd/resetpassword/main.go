// resetpassword sets a user's password by email or username (local DB recovery).
// Run from the REMI project root so ./data/trips.db resolves, or set SQLITE_PATH.
//
//	Usage: go run ./cmd/resetpassword <email-or-username>
//
// Interactive: prompts twice for new password (hidden when the terminal supports it).
// Non-interactive: set REMI_NEW_PASSWORD (e.g. CI); avoid shell history on shared machines.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
	_ "modernc.org/sqlite"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, `Reset a user's password (local SQLite only).

Usage:
  go run ./cmd/resetpassword <email-or-username>

Environment:
  SQLITE_PATH        Database file (default: ./data/trips.db)
  REMI_NEW_PASSWORD  New password (optional; if unset, you are prompted)

Run this on the machine that holds your database file, from the app directory
or with SQLITE_PATH pointing at your trips.db.
`)
		os.Exit(2)
	}

	identifier := strings.TrimSpace(os.Args[1])
	if identifier == "" {
		log.Fatal("email or username is required")
	}

	dbPath := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if dbPath == "" {
		dbPath = "./data/trips.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("database %s: %v", dbPath, err)
	}

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	var u trips.User
	if strings.Contains(identifier, "@") {
		u, err = repo.GetUserByEmail(ctx, identifier)
	} else {
		u, err = repo.GetUserByUsername(ctx, identifier)
	}
	if err != nil {
		log.Fatalf("user not found: %v", err)
	}

	newPass := strings.TrimSpace(os.Getenv("REMI_NEW_PASSWORD"))
	if newPass == "" {
		newPass = readNewPasswordInteractive()
	}
	if newPass == "" {
		log.Fatal("password is empty")
	}

	hash, err := trips.HashPassword(newPass)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}
	u.PasswordHash = hash
	if err := repo.UpdateUser(ctx, u); err != nil {
		log.Fatalf("update user: %v", err)
	}

	fmt.Printf("Password updated for %s (%s).\n", u.Username, u.Email)
	fmt.Println("You can log in with the new password. Existing sessions stay valid until they expire or you log out.")
}

func readNewPasswordInteractive() string {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		log.Fatal("stdin is not a terminal; set REMI_NEW_PASSWORD instead")
	}

	fmt.Fprint(os.Stderr, "New password: ")
	b1, err := term.ReadPassword(fd)
	if err != nil {
		log.Fatalf("read password: %v", err)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "Confirm new password: ")
	b2, err := term.ReadPassword(fd)
	if err != nil {
		log.Fatalf("read password: %v", err)
	}
	fmt.Fprintln(os.Stderr)

	p1 := strings.TrimSpace(string(b1))
	p2 := strings.TrimSpace(string(b2))
	if p1 != p2 {
		log.Fatal("passwords do not match")
	}
	return p1
}
