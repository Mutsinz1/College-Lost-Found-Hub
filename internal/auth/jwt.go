package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AppClaims are the claims carried by this application's session tokens.
type AppClaims struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

// IssueToken creates a signed session JWT for a user.
func IssueToken(secret string, claims AppClaims, ttl time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   claims.UserID.String(),
		"email": claims.Email,
		"role":  claims.Role,
		"iat":   now.Unix(),
		"exp":   now.Add(ttl).Unix(),
	})
	return token.SignedString([]byte(secret))
}

// ValidateToken parses and validates a session JWT and returns its claims.
func ValidateToken(secret, tokenString string) (*AppClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	sub, _ := mapClaims["sub"].(string)
	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, errors.New("invalid subject in token")
	}

	email, _ := mapClaims["email"].(string)
	role, _ := mapClaims["role"].(string)

	return &AppClaims{UserID: userID, Email: email, Role: role}, nil
}
