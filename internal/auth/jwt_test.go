package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueAndValidateToken(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := IssueToken(secret, AppClaims{
		UserID: userID,
		Email:  "student@college.edu",
		Role:   "user",
	}, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("IssueToken returned an empty token")
	}

	claims, err := ValidateToken(secret, token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Email != "student@college.edu" {
		t.Errorf("Email = %q, want %q", claims.Email, "student@college.edu")
	}
	if claims.Role != "user" {
		t.Errorf("Role = %q, want %q", claims.Role, "user")
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	token, err := IssueToken("secret-a", AppClaims{UserID: uuid.New(), Role: "user"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	if _, err := ValidateToken("secret-b", token); err == nil {
		t.Error("expected validation to fail with the wrong secret")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	token, err := IssueToken("secret", AppClaims{UserID: uuid.New(), Role: "user"}, -time.Minute)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	if _, err := ValidateToken("secret", token); err == nil {
		t.Error("expected validation to fail for an expired token")
	}
}

func TestValidateTokenGarbage(t *testing.T) {
	for _, tok := range []string{"", "not-a-jwt", "a.b.c", strings.Repeat("x", 500)} {
		if _, err := ValidateToken("secret", tok); err == nil {
			t.Errorf("expected validation to fail for garbage token %q", tok)
		}
	}
}
