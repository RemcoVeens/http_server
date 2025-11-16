package main

import (
	"net/http"

	"github.com/RemcoVeens/httpserver/internal/handlers"
)

func main() {
	servemux := http.NewServeMux()
	var apiC handlers.APIConfig
	servemux.Handle("/app/", http.StripPrefix("/app", apiC.MiddlewareMetricsInc(http.FileServer(http.Dir(".")))))
	servemux.HandleFunc("GET /api/healthz", handlers.HealthCodeHandler)
	servemux.HandleFunc("POST /api/validate_chirp", handlers.ChripHandler)
	servemux.HandleFunc("GET /admin/metrics", apiC.HitCounterHandler)
	servemux.HandleFunc("POST /admin/reset", apiC.Reset)
	server := http.Server{
		Handler: servemux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
