package database

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadDB() (*Queries, string, string) {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	secret := os.Getenv("SECRET")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Println("Could not open db: %w", err)
		os.Exit(1)
	}
	return New(db), platform, secret
}
