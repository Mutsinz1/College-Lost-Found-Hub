package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"lostfound/internal/auth"
	"lostfound/internal/config"
	"lostfound/internal/database"
)

// contextWithUser stores the authenticated user's ID and role in the context.
func contextWithUser(ctx context.Context, userID, role string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	return context.WithValue(ctx, RoleKey, role)
}

// RoleKey is the context key under which auth middleware stores the user role
const RoleKey contextKey = "user_role"

// AuthHandler serves login endpoints and issues session tokens.
type AuthHandler struct {
	repo     Store
	cfg      *config.Config
	verifier *auth.GoogleVerifier
}

// NewAuthHandler creates the authentication handler.
func NewAuthHandler(repo Store, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		repo:     repo,
		cfg:      cfg,
		verifier: auth.NewGoogleVerifier(cfg.Auth.GoogleClientID),
	}
}

type loginResponse struct {
	Token string        `json:"token"`
	User  database.User `json:"user"`
}

func (h *AuthHandler) issueSession(w http.ResponseWriter, user *database.User) {
	token, err := auth.IssueToken(h.cfg.JWT.Secret, auth.AppClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	}, h.cfg.JWT.Expiration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to issue session token", err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{Token: token, User: *user})
}

// emailDomainAllowed enforces the optional ALLOWED_EMAIL_DOMAIN restriction.
func (h *AuthHandler) emailDomainAllowed(email string) bool {
	domain := h.cfg.Auth.AllowedEmailDomain
	if domain == "" {
		return true
	}
	return strings.HasSuffix(strings.ToLower(email), "@"+strings.ToLower(domain))
}

// GoogleLogin verifies a Google ID token and returns an app session token.
// POST /api/auth/google  {"credential": "<google id token>"}
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Credential == "" {
		writeError(w, http.StatusBadRequest, "credential is required", nil)
		return
	}

	googleUser, err := h.verifier.Verify(req.Credential)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Google sign-in failed", err)
		return
	}

	if !h.emailDomainAllowed(googleUser.Email) {
		writeError(w, http.StatusForbidden, "This email domain is not allowed; use your school account", nil)
		return
	}

	user, err := h.repo.GetOrCreateUser(r.Context(), database.SSOUser{
		SSOID: "google:" + googleUser.Sub,
		Email: googleUser.Email,
		Name:  googleUser.Name,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to sign in", err)
		return
	}
	if !user.IsActive {
		writeError(w, http.StatusForbidden, "This account has been deactivated", nil)
		return
	}

	h.issueSession(w, user)
}

// DevLogin issues a session for an arbitrary identity. It is only mounted in
// the development environment and replaces the old unauthenticated /users/sso
// endpoint that trusted whatever the client sent.
// POST /api/auth/dev-login  {"email": "...", "name": "..."}
func (h *AuthHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "A valid email is required", nil)
		return
	}
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}

	user, err := h.repo.GetOrCreateUser(r.Context(), database.SSOUser{
		SSOID: "dev:" + strings.ToLower(req.Email),
		Email: strings.ToLower(req.Email),
		Name:  req.Name,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to sign in", err)
		return
	}

	h.issueSession(w, user)
}

// Middleware parses an optional Bearer session token and, when valid, stores
// the user's ID and role in the request context. Invalid tokens are rejected;
// missing tokens simply leave the request anonymous.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			if tokenString == header || tokenString == "" {
				writeError(w, http.StatusUnauthorized, "Invalid Authorization header", nil)
				return
			}

			claims, err := auth.ValidateToken(secret, tokenString)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "Invalid or expired session; sign in again", nil)
				return
			}

			ctx := r.Context()
			ctx = contextWithUser(ctx, claims.UserID.String(), claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth rejects requests that have no authenticated user.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(UserIDKey) == nil {
			writeError(w, http.StatusUnauthorized, "Sign in required", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects requests whose authenticated user is not an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(RoleKey).(string)
		if role != "admin" {
			writeError(w, http.StatusForbidden, "Admin access required", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
