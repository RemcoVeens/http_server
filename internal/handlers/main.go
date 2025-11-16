package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
)

type APIConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *APIConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safely increment the counter using Add(1).
		cfg.fileserverHits.Add(1)

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

func (cfg *APIConfig) Reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func (cfg *APIConfig) HitCounterHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write(fmt.Appendf(
		[]byte(""),
		fmt.Sprintf(
			"<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>",
			cfg.fileserverHits.Load(),
		),
	))
}

func ChripHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type input struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		Error      error  `json:"error"`
		CleandBody string `json:"cleaned_body"`
	}
	decoder := json.NewDecoder(r.Body)
	var params input
	status := 200
	respBody := returnVals{
		Error:      nil,
		CleandBody: "",
	}
	if err := decoder.Decode(&params); err != nil {
		respBody = returnVals{
			Error:      fmt.Errorf("Something went wrong"),
			CleandBody: "",
		}
		status = 400
	}
	log.Println(params.Body)
	if len(params.Body) > 140 {
		respBody = returnVals{
			Error:      fmt.Errorf("Chirp is too long"),
			CleandBody: "",
		}
		status = 400
	} else {
		var profaneWords = []string{"kerfuffle", "sharbert", "fornax"}
		var profanityPattern *regexp.Regexp
		wordPattern := strings.Join(profaneWords, "|")
		regexString := "(?i)\\b(" + wordPattern + ")\\b"
		profanityPattern = regexp.MustCompile(regexString)
		cleaned := profanityPattern.ReplaceAllString(params.Body, "****")
		respBody = returnVals{
			Error:      nil,
			CleandBody: cleaned,
		}
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		status = 500
		return
	}
	log.Println(respBody, "status:", status)
	w.WriteHeader(status)
	w.Write(dat)
}

func HealthCodeHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Println(err)
	}
}
