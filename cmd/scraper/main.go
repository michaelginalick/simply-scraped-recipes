package main

import (
	"cmp"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/michaelginalick/recipe-app/migrations"
	app "github.com/michaelginalick/recipe-app/pkg/application"
	"github.com/michaelginalick/recipe-app/pkg/store"
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
	s, err := store.New(*dbPath, migrations.SCHEMA)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	a, err := app.New(
		app.WithSourceLink(*source),
		app.WithThreadCount(*threadCount),
		app.WithRootHost(*rootHost),
		app.WithStore(s),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred: %v\n", err)
		os.Exit(1)
	}

	a.Run()
}
