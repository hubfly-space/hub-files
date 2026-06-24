package main

import (
	"flag"
	"fmt"
	"hubfly-files/internal/config"
	"hubfly-files/internal/server"
	"hubfly-files/internal/sessions"
	"hubfly-files/internal/sqlite"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func startServer(name, addr string, handler http.Handler) {
	go func() {
		log.Printf("%s starting on %s", name, addr)

		err := http.ListenAndServe(addr, handler)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("%s error: %v", name, err)
		}
	}()
}

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	cfg := config.LoadConfig()

	if *showVersion {
		fmt.Printf("hubfly-files %s\ncommit=%s\nbuildDate=%s\n", version, commit, buildDate)
		return
	}

	// Initialize db before building the server so search/indexing can be wired in.
	store, err := sqlite.New("hubfiles.sqlite")
	if err != nil {
		log.Printf("warning: database disabled: %v", err)
		store = nil
	}
	if store != nil {
		defer store.Close()
	}

	sessionStore := sessions.NewStore()
	srv := server.NewServer(cfg, sessionStore, store)
	defer srv.Close()

	//initialize server
	apiMux := srv.SetupRoutes()
	mgmtMux := srv.SetupManagementRoutes()

	startServer(
		"API Server",
		":"+cfg.APIPort,
		corsMiddleware(apiMux),
	)

	startServer(
		"Management Server",
		":"+cfg.ManagementPort,
		corsMiddleware(mgmtMux),
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Println("shutting down...")
	log.Println("servers stopped")
}
