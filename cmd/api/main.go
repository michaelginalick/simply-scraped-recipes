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

	"github.com/michaelginalick/recipe-app/internal/handler"
	"github.com/michaelginalick/recipe-app/internal/links"
	"github.com/michaelginalick/recipe-app/internal/middleware"
	"github.com/michaelginalick/recipe-app/internal/service"
	"github.com/michaelginalick/recipe-app/internal/sqlstore"
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
	maxJobs             = 2
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGSTOP)
	defer stop()

	dbReadPath := cmp.Or(os.Getenv("DB_PATH"), dbPath)
	readOnlyDBPath := fmt.Sprintf("file:%s?mode=ro&_busy_timeout=5000", dbReadPath)

	// Write connection must be opened first so the DB file exists before
	// the read-only connection attempts to open it.
	writeDB := openDB((fmt.Sprintf("file:%s?_busy_timeout=5000", dbReadPath)), maxWriteConnections)
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

	jobSvc := service.NewJobService(jobStore, sqlstore.NewSQLWriter(writeDB), maxJobs)
	jobSvc.Start(context.Background())

	linkClient, err := newLinkLinkClient()
	if err != nil {
		log.Fatal(err)
	}

	static, err := fs.Sub(staticFiles, assetsDirectory)
	if err != nil {
		log.Fatal(err)
	}

	mw := middleware.Chain{middleware.Recovery, middleware.MaxReader, middleware.Logging}
	jsonMW := middleware.Chain{middleware.Recovery, middleware.MaxReader, middleware.Logging, middleware.JSONContentType}

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.FS(static)))

	mux.HandleFunc("GET /r/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, static, "recipe.html")
	})

	mux.Handle("GET /recipe/{id}", jsonMW.ThenFunc(handler.GetHandler(reader)))
	mux.Handle("POST /recipe", mw.ThenFunc(handler.CreateHandler(recipeWriter)))

	mux.Handle("GET /search", jsonMW.ThenFunc(handler.SearchHandler(reader)))

	mux.Handle("POST /jobs", jsonMW.ThenFunc(handler.SubmitJobHandler(jobSvc)))
	mux.Handle("GET /jobs/{id}", jsonMW.ThenFunc(handler.GetJobHandler(reader)))

	mux.Handle("POST /probe", jsonMW.ThenFunc(handler.ProbeHandler(linkClient)))

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

func newLinkLinkClient() (*links.LinkClient, error) {
	return links.NewLinkClient(
		links.WithRoundTripper(
			&http.Transport{
				DialContext:           (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
				ResponseHeaderTimeout: 2 * time.Second,
			},
		),
	)
}
