package main

import (
	"fmt"
	"hubfly-files/internal/config"
	"hubfly-files/internal/server"
	"hubfly-files/internal/sessions"
	"log"
	"net/http"
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

func main() {
	cfg := config.LoadConfig()
	sessionStore := sessions.NewStore()
	srv := server.NewServer(cfg, sessionStore)

	// UI/API Server
	go func() {
		fmt.Printf("UI/API Server starting on :%s\n", cfg.APIPort)
		mux := srv.SetupRoutes()
		log.Fatal(http.ListenAndServe(":"+cfg.APIPort, corsMiddleware(mux)))
	}()

	// Management Server
	fmt.Printf("Management Server starting on :%s\n", cfg.ManagementPort)
	mux := srv.SetupManagementRoutes()
	log.Fatal(http.ListenAndServe(":"+cfg.ManagementPort, corsMiddleware(mux)))
}
