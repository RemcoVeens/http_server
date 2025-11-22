package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/RemcoVeens/httpserver/internal/auth"
	"github.com/RemcoVeens/httpserver/internal/database"
	"github.com/google/uuid"
)

type APIConfig struct {
	fileserverHits atomic.Int32
	Queries        *database.Queries
	Platform       string
	Secret         string
	PolkaKey       string
}

func (cfg *APIConfig) GetUserFromBearerToken(r *http.Request) (database.User, error) {
	tokn, err := auth.GetBearerToken(r.Header)
	user_id, err := auth.ValidateJWT(tokn, cfg.Secret)
	if err != nil {
		return database.User{}, fmt.Errorf("could not get user from token: %w", err)
	}
	return cfg.Queries.GetUserFromId(r.Context(), user_id)
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
func (cfg *APIConfig) UpdateUserHandel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	var params input
	status := 200
	if err := decoder.Decode(&params); err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error getting user input: %s", err))
		return
	}
	user, err := cfg.GetUserFromBearerToken(r)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error getting user: %s", err))
		return
	}
	hp, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "error hasing password: %s", err))
		return
	}
	if err = cfg.Queries.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hp,
		ID:             user.ID,
	}); err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "could not update user: %s", err))
		return
	}
	NewUser, err := cfg.Queries.GetUserFromEmail(r.Context(), params.Email)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "could not get updated user: %s", err))
		return
	}
	var m map[string]any
	tempJSON, _ := json.Marshal(NewUser)
	json.Unmarshal(tempJSON, &m)
	delete(m, "hashed_password")
	dat, err := json.MarshalIndent(m, "", "  ")
	log.Println(user.Email, "status:", status)
	w.WriteHeader(status)
	w.Write(dat)
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
	log.Println(user.Email, "has justy been created. status:", status)
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
	user, err := cfg.Queries.GetUserFromEmail(r.Context(), params.Email)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "user not found: %s", err))
		return
	}
	hp, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "could not hash : %s", err))
		return
	}
	ok, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "could not hash password: %s", err))
		return
	}
	if !ok {
		log.Print(user.HashedPassword)
		log.Print(hp)
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "could not authenticate: %s", err))
		return
	}
	jwtToken, err := auth.MakeJWT(user.ID, cfg.Secret, time.Duration(3600))
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error making jwt token: %s", err))
		return
	}
	tempJSON, _ := json.Marshal(user)
	RToken, _ := auth.MakeRefreshToken()
	RefToken, err := cfg.Queries.GenerateToken(r.Context(), database.GenerateTokenParams{
		Token:     RToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().AddDate(0, 0, 60),
		RevokedAt: sql.NullTime{Time: time.Time{}, Valid: false},
	})
	var m map[string]any
	json.Unmarshal(tempJSON, &m)
	delete(m, "hashed_password")
	m["token"] = jwtToken
	m["refresh_token"] = RefToken.Token
	dat, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		return
	}
	log.Println(params.Email, "just logged in")
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
	w.WriteHeader(status)
	w.Write(dat)
}
func (cfg *APIConfig) RemoveChirp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := 204
	chirpID := r.PathValue("chirpID")
	if chirpID == "" {
		w.WriteHeader(400)
		w.Write([]byte("please provide a chirp id to delete"))
		return
	}
	uuid, err := uuid.Parse(chirpID)
	if err != nil {
		w.WriteHeader(404)
		w.Write(fmt.Appendf([]byte(""), "could not process id: %v", err))
		return
	}
	chirp, err := cfg.Queries.GetChirpFromId(r.Context(), uuid)
	if err != nil {
		w.WriteHeader(404)
		w.Write(fmt.Appendf([]byte(""), "Error fetching chirps: %s", err))
		return
	}
	dat, err := json.Marshal(chirp)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		return
	}
	tokn, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error getting token: %s", err))
		return
	}
	user_id, err := auth.ValidateJWT(tokn, cfg.Secret)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error validation: %s", err))
		return
	}
	user, err := cfg.Queries.GetUserFromId(r.Context(), user_id)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error getting user from id: %s", err))
		return
	}
	if user.ID != chirp.UserID {
		w.WriteHeader(403)
		w.Write(fmt.Appendf([]byte(""), "This is not yours to delete"))
		return
	}
	err = cfg.Queries.DeleteChirpFromID(r.Context(), chirp.ID)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error deleting chirp: %s", err))
		return
	}
	w.WriteHeader(status)
	w.Write(dat)
}
func (cfg *APIConfig) Chirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type input struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	var params input
	status := 201
	if err := decoder.Decode(&params); err != nil {
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		w.WriteHeader(500)
		return
	}
	token_string, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(403)
		w.Write(fmt.Appendf([]byte(""), "Could not get token: %s", err))
		return
	}
	UserID, err := auth.ValidateJWT(token_string, cfg.Secret)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Validation error: %s", err))
		return
	}

	user, err := cfg.Queries.GetUserFromId(r.Context(), UserID)
	if err != nil {
		log.Printf("Could not get user from id: %s", UserID)
	}
	chirp, err := cfg.Queries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   params.Body,
		UserID: user.ID,
	})
	if err != nil {
		log.Printf("Could not create chirp (%s) from user: %s", params.Body, UserID)
	}
	dat, err := json.Marshal(chirp)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		return
	}
	w.WriteHeader(status)
	w.Write(dat)
}
func (cfg *APIConfig) RefreshHandel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := 200
	user, err := cfg.GetUserFromBearerToken(r)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error getting user: %s", err))
		return
	}
	w.WriteHeader(status)
	type response struct {
		Token string `json:"token"`
	}
	new_token, err := auth.MakeJWT(user.ID, cfg.Secret, 3600)
	dat, err := json.Marshal(response{
		Token: new_token,
	})
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error marshalling JSON: %s", err))
		return
	}
	w.Write(dat)
}
func (cfg *APIConfig) RevokeHandel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokn, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error getting token: %s", err))
		return
	}
	err = cfg.Queries.RevokeToken(r.Context(), tokn)
	if err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "could not revoke token: %s", err))
		return
	}
	w.WriteHeader(204)
}
func (cfg *APIConfig) PolkaWebhook(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	var params input
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		w.WriteHeader(500)
		w.Write(fmt.Appendf([]byte(""), "Error paring input: %s", err))
		return
	}
	polkaApiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write(fmt.Appendf([]byte(""), "Error getting API key: %s", err))
		return
	}
	log.Printf("%s and %s", polkaApiKey, cfg.PolkaKey)
	if polkaApiKey != cfg.PolkaKey {
		w.WriteHeader(401)
		w.Write([]byte("Invalid API key"))
		return
	}
	switch params.Event {
	case "user.upgraded":
		userID, err := uuid.Parse(params.Data.UserID)
		if err != nil {
			w.WriteHeader(500)
			w.Write(fmt.Appendf([]byte(""), "Error parsing user ID: %s", err))
			return
		}
		user, err := cfg.Queries.GetUserFromId(r.Context(), userID)
		if err != nil {
			w.WriteHeader(404)
			w.Write(fmt.Appendf([]byte(""), "Error getting user: %s", err))
			return
		}
		err = cfg.Queries.UpgradeUserFromID(r.Context(), user.ID)
		if err != nil {
			w.WriteHeader(500)
			w.Write(fmt.Appendf([]byte(""), "Error upgrading user: %s", err))
			return
		}
		log.Printf("upgraded %s to red!", user.Email)
		w.WriteHeader(204)
		w.Write(fmt.Appendf([]byte(""), "user(%s) has been upgrade", user.Email))
		return
	default:
		w.WriteHeader(204)
		w.Write(fmt.Appendf([]byte(""), "Unknown event: %s", params.Event))
		return

	}
}
func HealthCodeHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Println(err)
	}
}
