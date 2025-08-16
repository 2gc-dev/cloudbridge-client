package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateJWT(secret string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func TestValidateJWTToken_Valid(t *testing.T) {
	secret := "testsecret"
	claims := jwt.MapClaims{
		"sub": "user1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, err := generateJWT(secret, claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	am, err := NewAuthManager(&AuthConfig{Type: "jwt", Secret: secret})
	if err != nil {
		t.Fatalf("failed to create auth manager: %v", err)
	}
	_, err = am.ValidateToken(tokenStr)
	if err != nil {
		t.Errorf("expected valid token, got error: %v", err)
	}
}

func TestValidateJWTToken_Expired(t *testing.T) {
	secret := "testsecret"
	claims := jwt.MapClaims{
		"sub": "user1",
		"exp": time.Now().Add(-time.Hour).Unix(),
	}
	tokenStr, err := generateJWT(secret, claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	am, err := NewAuthManager(&AuthConfig{Type: "jwt", Secret: secret})
	if err != nil {
		t.Fatalf("failed to create auth manager: %v", err)
	}
	_, err = am.ValidateToken(tokenStr)
	if err == nil {
		t.Errorf("expected error for expired token, got nil")
	}
}

func TestValidateJWTToken_InvalidSignature(t *testing.T) {
	secret := "testsecret"
	claims := jwt.MapClaims{
		"sub": "user1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, err := generateJWT("wrongsecret", claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	am, err := NewAuthManager(&AuthConfig{Type: "jwt", Secret: secret})
	if err != nil {
		t.Fatalf("failed to create auth manager: %v", err)
	}
	_, err = am.ValidateToken(tokenStr)
	if err == nil {
		t.Errorf("expected error for invalid signature, got nil")
	}
}

func TestValidateJWTToken_NoSubClaim(t *testing.T) {
	secret := "testsecret"
	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr, err := generateJWT(secret, claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	am, err := NewAuthManager(&AuthConfig{Type: "jwt", Secret: secret})
	if err != nil {
		t.Fatalf("failed to create auth manager: %v", err)
	}
	tok, err := am.ValidateToken(tokenStr)
	if err != nil {
		t.Errorf("expected valid token, got error: %v", err)
	}
	_, err = am.ExtractSubject(tok)
	if err == nil {
		t.Errorf("expected error for missing sub claim, got nil")
	}
} 