package auth

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/2gc-dev/cloudbridge-client/pkg/errors"
)

// AuthManager handles authentication with relay server
type AuthManager struct {
	config     *AuthConfig
	jwtSecret  []byte
	publicKey  *rsa.PublicKey
	httpClient *http.Client
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Type     string         `json:"type"`
	Secret   string         `json:"secret"`
	Keycloak *KeycloakConfig `json:"keycloak,omitempty"`
}

// KeycloakConfig contains Keycloak-specific configuration
type KeycloakConfig struct {
	ServerURL string `json:"server_url"`
	Realm     string `json:"realm"`
	ClientID  string `json:"client_id"`
	JWKSURL   string `json:"jwks_url"`
}

// JWKS represents JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthConfig) (*AuthManager, error) {
	am := &AuthManager{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	switch config.Type {
	case "jwt":
		if config.Secret == "" {
			return nil, fmt.Errorf("JWT secret is required")
		}
		am.jwtSecret = []byte(config.Secret)

	case "keycloak":
		if config.Keycloak == nil {
			return nil, fmt.Errorf("Keycloak configuration is required")
		}
		if err := am.setupKeycloak(); err != nil {
			return nil, fmt.Errorf("failed to setup Keycloak: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported authentication type: %s", config.Type)
	}

	return am, nil
}

// setupKeycloak initializes Keycloak authentication
func (am *AuthManager) setupKeycloak() error {
	if am.config.Keycloak.JWKSURL == "" {
		am.config.Keycloak.JWKSURL = fmt.Sprintf(
			"%s/realms/%s/protocol/openid-connect/certs",
			am.config.Keycloak.ServerURL,
			am.config.Keycloak.Realm,
		)
	}

	// Fetch JWKS
	jwks, err := am.fetchJWKS()
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Convert first key to RSA public key
	if len(jwks.Keys) == 0 {
		return fmt.Errorf("no keys found in JWKS")
	}

	key := jwks.Keys[0]
	publicKey, err := am.jwkToRSAPublicKey(key)
	if err != nil {
		return fmt.Errorf("failed to convert JWK to RSA public key: %w", err)
	}

	am.publicKey = publicKey
	return nil
}

// fetchJWKS fetches JSON Web Key Set from Keycloak
func (am *AuthManager) fetchJWKS() (*JWKS, error) {
	resp, err := am.httpClient.Get(am.config.Keycloak.JWKSURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JWKS: %s", resp.Status)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	return &jwks, nil
}

// jwkToRSAPublicKey converts JWK to RSA public key
func (am *AuthManager) jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// This is a simplified implementation
	// In production, you should use a proper JWK library
	return nil, fmt.Errorf("JWK to RSA conversion not implemented")
}

// ValidateToken validates a JWT token
func (am *AuthManager) ValidateToken(tokenString string) (*jwt.Token, error) {
	switch am.config.Type {
	case "jwt":
		return am.validateJWTToken(tokenString)
	case "keycloak":
		return am.validateKeycloakToken(tokenString)
	default:
		return nil, fmt.Errorf("unsupported authentication type")
	}
}

// validateJWTToken validates a JWT token with HMAC
func (am *AuthManager) validateJWTToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, errors.NewRelayError(errors.ErrInvalidToken, fmt.Sprintf("JWT validation failed: %v", err))
	}

	if !token.Valid {
		return nil, errors.NewRelayError(errors.ErrInvalidToken, "Invalid JWT token")
	}

	return token, nil
}

// validateKeycloakToken validates a Keycloak token
func (am *AuthManager) validateKeycloakToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.publicKey, nil
	})

	if err != nil {
		return nil, errors.NewRelayError(errors.ErrInvalidToken, fmt.Sprintf("Keycloak token validation failed: %v", err))
	}

	if !token.Valid {
		return nil, errors.NewRelayError(errors.ErrInvalidToken, "Invalid Keycloak token")
	}

	// Validate claims
	if err := am.validateKeycloakClaims(token.Claims); err != nil {
		return nil, errors.NewRelayError(errors.ErrInvalidToken, fmt.Sprintf("Invalid claims: %v", err))
	}

	return token, nil
}

// validateKeycloakClaims validates Keycloak-specific claims
func (am *AuthManager) validateKeycloakClaims(claims jwt.Claims) error {
	// Validate issuer
	if issuer, ok := claims.(jwt.MapClaims)["iss"]; ok {
		expectedIssuer := fmt.Sprintf("%s/realms/%s", am.config.Keycloak.ServerURL, am.config.Keycloak.Realm)
		if issuer != expectedIssuer {
			return fmt.Errorf("invalid issuer: expected %s, got %s", expectedIssuer, issuer)
		}
	}

	// Validate audience
	if aud, ok := claims.(jwt.MapClaims)["aud"]; ok {
		if aud != am.config.Keycloak.ClientID {
			return fmt.Errorf("invalid audience: expected %s, got %s", am.config.Keycloak.ClientID, aud)
		}
	}

	return nil
}

// ExtractSubject extracts subject from token
func (am *AuthManager) ExtractSubject(token *jwt.Token) (string, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	subject, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("subject claim not found or invalid")
	}

	return subject, nil
}

// CreateAuthMessage creates an authentication message for relay server
func (am *AuthManager) CreateAuthMessage(tokenString string) (map[string]interface{}, error) {
	// Validate token first
	token, err := am.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Extract subject for rate limiting
	subject, err := am.ExtractSubject(token)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type":  "auth",
		"token": tokenString,
		"sub":   subject,
	}, nil
}

// GetTokenFromHeader extracts token from Authorization header
func (am *AuthManager) GetTokenFromHeader(header string) (string, error) {
	if header == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	parts := strings.Split(header, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
} 