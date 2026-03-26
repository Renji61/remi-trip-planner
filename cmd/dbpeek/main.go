// dbpeek prints UI flags for a trip (one-off debugging). Usage: go run ./cmd/dbpeek <tripID>
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/dbpeek <tripID>")
	}
	id := os.Args[1]
	dbPath := "./data/trips.db"
	if p := os.Getenv("SQLITE_PATH"); p != "" {
		dbPath = p
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var ss, sv, sf, sp int
	var name string
	err = db.QueryRowContext(context.Background(),
		`SELECT name, ui_show_stay, ui_show_vehicle, ui_show_flights, ui_show_spends FROM trips WHERE id = ?`, id,
	).Scan(&name, &ss, &sv, &sf, &sp)
	if err == sql.ErrNoRows {
		fmt.Println("no row for id", id)
		os.Exit(1)
	}
	if err != nil {
		// Column might be missing on very old DB
		log.Fatal(err)
	}
	fmt.Printf("trip %s (%q)\n", id, name)
	fmt.Printf("ui_show_stay=%d vehicle=%d flights=%d spends=%d\n", ss, sv, sf, sp)
}
