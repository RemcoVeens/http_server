package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safely increment the counter using Add(1).
		cfg.fileserverHits.Add(1)

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

func HealthCodeHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Println(err)
	}
}
func (cfg *apiConfig) Reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) HitCounterHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	_, err := w.Write(fmt.Appendf([]byte(""), "Hits: %v\n", cfg.fileserverHits.Load()))
	if err != nil {
		log.Println(err)
	}
}

func main() {
	servemux := http.NewServeMux()
	var apiC apiConfig
	servemux.Handle("/app/", http.StripPrefix("/app", apiC.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	servemux.HandleFunc("GET /healthz", HealthCodeHandler)
	servemux.HandleFunc("GET /metrics", apiC.HitCounterHandler)
	servemux.HandleFunc("POST /reset", apiC.Reset)
	server := http.Server{
		Handler: servemux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
