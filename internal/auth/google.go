package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

// GoogleUser is the identity extracted from a verified Google ID token.
type GoogleUser struct {
	Sub          string // Google's stable user ID
	Email        string
	Name         string
	HostedDomain string // "hd" claim, set for Google Workspace accounts
}

// GoogleVerifier verifies Google ID tokens against Google's published JWKS.
type GoogleVerifier struct {
	ClientID string

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	client    *http.Client
}

// NewGoogleVerifier creates a verifier for the given OAuth client ID.
func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{
		ClientID: clientID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

// getKey returns the RSA public key for a key ID, refreshing the JWKS cache
// when the key is unknown or the cache is older than an hour.
func (v *GoogleVerifier) getKey(kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if key, ok := v.keys[kid]; ok && time.Since(v.fetchedAt) < time.Hour {
		return key, nil
	}

	resp, err := v.client.Get(googleJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Google keys: status %d", resp.StatusCode)
	}

	var set jwks
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, fmt.Errorf("failed to decode Google keys: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}
	}

	v.keys = keys
	v.fetchedAt = time.Now()

	key, ok := v.keys[kid]
	if !ok {
		return nil, errors.New("unknown key ID in Google token")
	}
	return key, nil
}

// Verify checks a Google ID token's signature, issuer, audience and expiry,
// and returns the identity it asserts.
func (v *GoogleVerifier) Verify(idToken string) (*GoogleUser, error) {
	if v.ClientID == "" {
		return nil, errors.New("google sign-in is not configured (GOOGLE_CLIENT_ID missing)")
	}

	token, err := jwt.Parse(idToken, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token missing key ID")
		}
		return v.getKey(kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithAudience(v.ClientID), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("invalid Google token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid Google token claims")
	}

	iss, _ := claims["iss"].(string)
	if iss != "accounts.google.com" && iss != "https://accounts.google.com" {
		return nil, errors.New("invalid token issuer")
	}

	if verified, ok := claims["email_verified"].(bool); ok && !verified {
		return nil, errors.New("Google account email is not verified")
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	if sub == "" || email == "" {
		return nil, errors.New("Google token missing identity claims")
	}

	name, _ := claims["name"].(string)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	hd, _ := claims["hd"].(string)

	return &GoogleUser{Sub: sub, Email: email, Name: name, HostedDomain: hd}, nil
}
