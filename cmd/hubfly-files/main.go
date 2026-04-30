package main

import (
	"fmt"
	"hubfly-files/internal/config"
	"log"
	"net/http"
)

func main() {
	cfg := config.LoadConfig()

	// UI/API Server (Port 8080 by default)
	go func() {
		fmt.Printf("UI/API Server starting on :%s\n", cfg.APIPort)
		mux := http.NewServeMux()
		mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OK"))
		})
		log.Fatal(http.ListenAndServe(":"+cfg.APIPort, mux))
	}()

	// Management Server (Port 9090 by default)
	fmt.Printf("Management Server starting on :%s\n", cfg.ManagementPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/sessions/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	log.Fatal(http.ListenAndServe(":"+cfg.ManagementPort, mux))
}
