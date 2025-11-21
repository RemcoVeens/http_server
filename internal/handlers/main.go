package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/RemcoVeens/httpserver/internal/auth"
	"github.com/RemcoVeens/httpserver/internal/database"
	"github.com/google/uuid"
)

type APIConfig struct {
	fileserverHits atomic.Int32
	Queries        *database.Queries
	Platform       string
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
	if cfg.Platform != "dev" {
		w.WriteHeader(401)
		return
	}
	cfg.fileserverHits.Store(0)
	if err := cfg.Queries.DeleteAllUsers(r.Context()); err != nil {
		log.Fatalln("could not reset users: %w", err)
	}
}
func (cfg *APIConfig) HitCounterHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write(fmt.Appendf(
		[]byte(""),
		"<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>",
		cfg.fileserverHits.Load(),
	))
}
func (cfg *APIConfig) CreateUserHandel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	var params input
	var user database.User
	status := 201
	if err := decoder.Decode(&params); err != nil {
		user = database.User{}
		status = 400
	}
	log.Println(params.Email)
	pass, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("could not hash password: %s", err)))
		return
	}
	user, err = cfg.Queries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: pass,
	})
	if err != nil {
		log.Printf("could not create user: %s", err)
	}

	tempJSON, _ := json.Marshal(user)
	var m map[string]interface{}
	json.Unmarshal(tempJSON, &m)
	delete(m, "hashed_password")
	dat, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error marshalling JSON: %s", err)))
		return
	}
	log.Println(user, "status:", status)
	w.WriteHeader(status)
	w.Write(dat)
}
func (cfg *APIConfig) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	var params input
	var user database.User
	status := 200
	if err := decoder.Decode(&params); err != nil {
		user = database.User{}
		status = 400
	}
	log.Println(params.Email)
	user, err := cfg.Queries.GetUserFromEmail(r.Context(), params.Email)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("user not found: %s", err)))
		return
	}
	hp, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("could not hash : %s", err)))
		return
	}
	ok, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("could not hash password: %s", err)))
		return
	}
	if !ok {
		log.Print(user.HashedPassword)
		log.Print(hp)
		w.WriteHeader(401)
		w.Write([]byte(fmt.Sprintf("could not authenticate: %s", err)))
		return
	}
	tempJSON, _ := json.Marshal(user)
	var m map[string]interface{}
	json.Unmarshal(tempJSON, &m)
	delete(m, "hashed_password")
	dat, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Error marshalling JSON: %s", err)))
		return
	}
	log.Println(user, "status:", status)
	w.WriteHeader(status)
	w.Write(dat)
}
func (cfg *APIConfig) GetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.Queries.GetChirps(r.Context())
	status := 200
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error fetching chirps: %s", err))
		return
	}
	dat, err := json.Marshal(chirps)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		return
	}
	log.Println(chirps, "status:", status)
	w.WriteHeader(status)
	w.Write(dat)
}
func (cfg *APIConfig) GetChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("chirp_id")
	if chirpID == "" {
		w.WriteHeader(400)
		w.Write([]byte("please provide a chirp id to fetch"))
		return
	}
	uuid, err := uuid.Parse(chirpID)
	if err != nil {
		w.WriteHeader(404)
		w.Write(fmt.Appendf([]byte(""), "could not make id from parameter: %s", err))
		return
	}
	chirps, err := cfg.Queries.GetChirpFromId(r.Context(), uuid)
	status := 200
	if err != nil {
		w.WriteHeader(404)
		w.Write(fmt.Appendf([]byte(""), "Error fetching chirps: %s", err))
		return
	}
	dat, err := json.Marshal(chirps)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		return
	}
	log.Println(chirps, "status:", status)
	w.WriteHeader(status)
	w.Write(dat)
}
func (cfg *APIConfig) Chirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type input struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	var params input
	status := 201
	if err := decoder.Decode(&params); err != nil {
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		w.WriteHeader(500)
		return
	}
	user, err := cfg.Queries.GetUserFromId(r.Context(), params.UserID)
	if err != nil {
		log.Printf("Could not get user from id: %s", params.UserID)
	}
	chirp, err := cfg.Queries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   params.Body,
		UserID: user.ID,
	})
	if err != nil {
		log.Printf("Could not create chirp (%s) from user: %s", params.Body, params.UserID)
	}
	dat, err := json.Marshal(chirp)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		return
	}
	log.Println(chirp, "status:", status)
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
