package main

import (
	"cmp"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/michaelginalick/recipe-app/cmd/recipe-api/handler"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/middleware"
	srv "github.com/michaelginalick/recipe-app/cmd/recipe-api/service"
	"github.com/michaelginalick/recipe-app/cmd/recipe-api/sqlstore"
	"github.com/michaelginalick/recipe-app/migrations"
	"github.com/michaelginalick/recipe-app/pkg/links"
	pkgstore "github.com/michaelginalick/recipe-app/pkg/store"
	_ "modernc.org/sqlite"
)

//go:embed static
var staticFiles embed.FS

const (
	maxReadConnections  = 10
	maxWriteConnections = 1
	assetsDirectory     = "static"
	dbPath              = "./recipes.db"
	maxJobs             = 2
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGSTOP)
	defer stop()

	dbReadPath := cmp.Or(os.Getenv("DB_PATH"), dbPath)
	readOnlyDBPath := fmt.Sprintf("file:%s?mode=ro", dbReadPath)

	// Write connection must be opened first so the DB file exists before
	// the read-only connection attempts to open it.
	writeDB := openDB(dbReadPath, maxWriteConnections)
	defer writeDB.Close()

	if err := migrations.Run(writeDB); err != nil {
		log.Fatal("migrations: ", err)
	}

	readDB := openDB(readOnlyDBPath, maxReadConnections)
	defer readDB.Close()

	reader := sqlstore.NewSQLReader(readDB)
	recipeWriter := sqlstore.NewSQLWriter(writeDB)
	jobStore := sqlstore.NewSQLJobStore(writeDB)

	if err := jobStore.RecoverJobs(ctx); err != nil {
		log.Fatal("recover jobs: ", err)
	}

	jobSvc := srv.NewJobService(jobStore, pkgstore.NewFromDB(writeDB), maxJobs)
	jobSvc.Start(context.Background())

	static, err := fs.Sub(staticFiles, assetsDirectory)
	if err != nil {
		log.Fatal(err)
	}

	mw := middleware.Chain{middleware.Recovery, middleware.MaxReader, middleware.Logging}
	jsonMW := middleware.Chain{middleware.Recovery, middleware.MaxReader, middleware.Logging, middleware.JSONContentType}

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.FS(static)))

	mux.Handle("GET /recipe/{id}", jsonMW.ThenFunc(handler.GetHandler(reader)))
	mux.Handle("POST /recipe", mw.ThenFunc(handler.CreateHandler(recipeWriter)))

	mux.Handle("GET /search", jsonMW.ThenFunc(handler.SearchHandler(reader)))

	mux.Handle("POST /jobs", jsonMW.ThenFunc(handler.SubmitJobHandler(jobSvc)))
	mux.Handle("GET /jobs/{id}", jsonMW.ThenFunc(handler.GetJobHandler(reader)))

	mux.Handle("POST /probe", jsonMW.ThenFunc(handler.ProbeHandler(srv.NewProbeService(links.NewLinkClient(&http.Transport{
		DialContext:           (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
		ResponseHeaderTimeout: 2 * time.Second,
	})))))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

	jobSvc.Stop()
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

	return db
}
