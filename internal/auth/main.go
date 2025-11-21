package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)

}
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn * time.Second)),
		Subject:   userID.String(),
	})
	return tkn.SignedString([]byte(tokenSecret))
}

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, errors.New("invalid or expired token")
	}
	userID, err := uuid.Parse(claims.RegisteredClaims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UserID format in token claims: %w", err)
	}
	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	key := headers.Get("Authorization")
	if key == "" {
		return key, fmt.Errorf("field was empty/not found")
	}
	key = strings.Join(strings.Split(key, " ")[1:], " ")
	return key, nil

}
