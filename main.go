package main

import (
	"net/http"

	"github.com/RemcoVeens/httpserver/internal/database"
	"github.com/RemcoVeens/httpserver/internal/handlers"

	_ "github.com/lib/pq"
)

func main() {
	var apiC handlers.APIConfig
	apiC.Queries, apiC.Platform, apiC.Secret = database.LoadDB()
	servemux := http.NewServeMux()
	servemux.Handle("/app/", http.StripPrefix("/app", apiC.MiddlewareMetricsInc(http.FileServer(http.Dir(".")))))
	servemux.HandleFunc("GET /api/healthz", handlers.HealthCodeHandler)
	servemux.HandleFunc("POST /api/users", apiC.CreateUserHandel)
	servemux.HandleFunc("PUT /api/users", apiC.UpdateUserHandel)
	servemux.HandleFunc("POST /api/login", apiC.LoginHandler)
	servemux.HandleFunc("GET /admin/metrics", apiC.HitCounterHandler)
	servemux.HandleFunc("GET /api/chirps", apiC.GetChirps)
	servemux.HandleFunc("GET /api/chirps/{chirp_id}", apiC.GetChirp)
	servemux.HandleFunc("DELETE /api/chirps/{chirpID}", apiC.RemoveChirp)
	servemux.HandleFunc("POST /api/refresh", apiC.RefreshHandel)
	servemux.HandleFunc("POST /api/revoke", apiC.RevokeHandel)
	servemux.HandleFunc("POST /api/chirps", apiC.Chirps)
	servemux.HandleFunc("POST /admin/reset", apiC.Reset)
	server := http.Server{
		Handler: servemux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
