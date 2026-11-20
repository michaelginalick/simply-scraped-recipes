package main

import (
	"cmp"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	app "github.com/michaelginalick/recipe-app/internal/application"
	"github.com/michaelginalick/recipe-app/internal/sqlstore"
	"github.com/michaelginalick/recipe-app/migrations"
	_ "modernc.org/sqlite"
)

func main() {
	source := flag.String("source", "https://theeburgerdude.com/lentil-mushroom-ground-beef/", "starting link")
	threadCount := flag.Int("concurrency", 3, "number of active threads. Max 10, min 1")
	dbPath := flag.String("db", cmp.Or(os.Getenv("DB_PATH"), "./recipes.db"), "path to SQLite database file")
	debug := flag.Bool("debug", false, "enable debug logging")
	rootHost := flag.String("rootHost", "theeburgerdude.com", "root of site being scraped")

	flag.Parse()

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	db, err := openDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := migrations.Run(db); err != nil {
		log.Fatal("migrations: ", err)
	}

	a, err := app.New(
		app.WithSourceLink(*source),
		app.WithThreadCount(*threadCount),
		app.WithRootHost(*rootHost),
		app.WithStore(sqlstore.NewSQLWriter(db)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred: %v\n", err)
		os.Exit(1)
	}

	a.Run()
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
