package auth_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/RemcoVeens/httpserver/internal/auth" // Assuming the code is in the "auth" package

	"github.com/google/uuid"
)

// TestHashPassword tests the password hashing function.
func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == password {
		t.Errorf("Hash should not be equal to the original password")
	}

	if len(hash) < 60 {
		t.Errorf("Generated hash seems too short: %d", len(hash))
	}
}

// TestCheckPasswordHash tests the password comparison function.
func TestCheckPasswordHash(t *testing.T) {
	password := "secret_password"
	wrongPassword := "wrong_password"

	// 1. Generate a valid hash
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to generate hash for testing: %v", err)
	}

	// Test Case 1: Correct Password
	match, err := auth.CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed for correct password: %v", err)
	}
	if !match {
		t.Errorf("Expected correct password to match, but it did not")
	}

	// Test Case 2: Incorrect Password
	match, err = auth.CheckPasswordHash(wrongPassword, hash)
	if err != nil {
		// Note: argon2id.ComparePasswordAndHash usually only returns an error on internal failure,
		// not on mismatch, but we handle it just in case.
		t.Fatalf("CheckPasswordHash failed for wrong password: %v", err)
	}
	if match {
		t.Errorf("Expected wrong password NOT to match, but it did")
	}
}

// TestJWT tests the combined functionality of MakeJWT and ValidateJWT.
func TestJWT(t *testing.T) {
	userID := uuid.New()
	secret := "secure-testing-secret-key"
	expiresIn := time.Minute * 5

	// Test Case 1: Successful creation and validation
	tokenString, err := auth.MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	validatedID, err := auth.ValidateJWT(tokenString, secret)
	if err != nil {
		t.Errorf("ValidateJWT failed with correct secret: %v", err)
	}
	if validatedID != userID {
		t.Errorf("Validated UserID (%s) does not match original UserID (%s)", validatedID, userID)
	}

	// Test Case 2: Validation with wrong secret
	wrongSecret := "incorrect-secret-key"
	_, err = auth.ValidateJWT(tokenString, wrongSecret)
	if err == nil {
		t.Errorf("ValidateJWT succeeded with wrong secret, expected failure")
	}

	// Test Case 3: Validation of an expired token
	shortExpiresIn := time.Millisecond * 50
	expiredToken, err := auth.MakeJWT(userID, secret, shortExpiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed for expiration test: %v", err)
	}

	// Wait for the token to expire. We multiply by 2 to ensure expiration passes the grace period.
	time.Sleep(shortExpiresIn * 2)

	_, err = auth.ValidateJWT(expiredToken, secret)
	if err == nil {
		t.Errorf("ValidateJWT succeeded with an expired token, expected failure")
	} else {
		// The error from jwt.ParseWithClaims is often a composite error (e.g., "token has invalid claims: token is expired").
		// We check if the error string contains the key phrase "token is expired" for robustness.
		expectedErrorPart := "token is expired"
		errStr := err.Error()
		if !strings.Contains(errStr, expectedErrorPart) {
			t.Errorf("ValidateJWT failed with unexpected error for expired token. Expected error to contain '%s', got: %s", expectedErrorPart, errStr)
		}
	}
}

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	secret_key := "pindas"
	toke := fmt.Sprintf("Bearer %s", secret_key)
	_, err := auth.GetBearerToken(headers)
	if err == nil {
		t.Fatalf("did not not get auth key")
	}
	headers.Add("Authorization", toke)
	key, _ := auth.GetBearerToken(headers)
	if key != secret_key {
		t.Fatalf("token is not %v", secret_key)
	}
}
