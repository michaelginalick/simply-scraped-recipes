package main

import (
	"cmp"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/handler"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/middleware"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/sqlstore"
	"github.com/michaelginalick/recipe-app/migrations"
	_ "modernc.org/sqlite"
)

//go:embed static
var staticFiles embed.FS

const (
	maxReadConnections  = 10
	maxWriteConnections = 1
	assetsDirectory     = "static"
	dbPath              = "./recipes.db"
)

func main() {
	dbReadPath := cmp.Or(os.Getenv("DB_PATH"), dbPath)
	readOnlyDBPath := fmt.Sprintf("file:%s?mode=ro", dbReadPath)

	// Write connection must be opened first so the DB file exists before
	// the read-only connection attempts to open it.
	writeDB := openDB(dbReadPath, maxWriteConnections)
	if writeDB == nil {
		log.Fatal("failed to open write db")
	}

	defer writeDB.Close()

	readDB := openDB(readOnlyDBPath, maxReadConnections)
	if readDB == nil {
		log.Fatal("failed to open read db")
	}
	defer readDB.Close()

	recipeReader := sqlstore.NewSQLReader(readDB)
	recipeWriter := sqlstore.NewSQLWriter(writeDB)

	static, err := fs.Sub(staticFiles, assetsDirectory)
	if err != nil {
		log.Fatal(err)
	}

	mw := middleware.Chain{middleware.Recovery, middleware.Logging}
	jsonMW := middleware.Chain{middleware.Recovery, middleware.Logging, middleware.JSONContentType}

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.Handle("GET /recipe/{id}", jsonMW.ThenFunc(handler.GetHandler(recipeReader)))
	mux.Handle("POST /recipe", mw.ThenFunc(handler.CreateHandler(recipeWriter)))
	mux.Handle("GET /search", jsonMW.ThenFunc(handler.SearchHandler(recipeReader)))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func openDB(dsn string, maxConns int) *sql.DB {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if schemaExists(db) {
		return db
	}

	if err := applySchema(db, migrations.SCHEMA); err != nil {
		db.Close()
		return nil
	}

	return db
}

func applySchema(db *sql.DB, schema string) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	return nil
}

func schemaExists(db *sql.DB) bool {
	exists, err := sqlstore.SchemaExists(db)
	if err != nil {
		log.Fatal(err)
	}
	return exists
}
