package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func rateLimitedRouter(limit int) http.Handler {
	r := chi.NewRouter()
	r.With(RateLimit(limit, time.Hour)).Post("/write", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	r.Get("/read", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}

func call(t *testing.T, h http.Handler, method, path, ip string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = ip + ":12345"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestRateLimitBlocksAfterTheLimit(t *testing.T) {
	h := rateLimitedRouter(3)

	for i := 1; i <= 3; i++ {
		if code := call(t, h, http.MethodPost, "/write", "203.0.113.10"); code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201", i, code)
		}
	}
	if code := call(t, h, http.MethodPost, "/write", "203.0.113.10"); code != http.StatusTooManyRequests {
		t.Errorf("request 4: status = %d, want 429", code)
	}
}

func TestRateLimitIsPerClientIP(t *testing.T) {
	h := rateLimitedRouter(2)

	for i := 0; i < 2; i++ {
		call(t, h, http.MethodPost, "/write", "203.0.113.10")
	}
	if code := call(t, h, http.MethodPost, "/write", "203.0.113.10"); code != http.StatusTooManyRequests {
		t.Fatalf("first client should be throttled, got %d", code)
	}

	// A different address must be unaffected: one noisy client must not lock
	// everyone else out.
	if code := call(t, h, http.MethodPost, "/write", "203.0.113.99"); code != http.StatusCreated {
		t.Errorf("second client: status = %d, want 201", code)
	}
}

func TestRateLimitLeavesReadsAlone(t *testing.T) {
	h := rateLimitedRouter(1)

	call(t, h, http.MethodPost, "/write", "203.0.113.10")
	if code := call(t, h, http.MethodPost, "/write", "203.0.113.10"); code != http.StatusTooManyRequests {
		t.Fatalf("writes should be throttled, got %d", code)
	}

	// Browsing is not an abuse vector and must never be limited.
	for i := 0; i < 20; i++ {
		if code := call(t, h, http.MethodGet, "/read", "203.0.113.10"); code != http.StatusOK {
			t.Fatalf("read %d: status = %d, want 200", i, code)
		}
	}
}

func TestRateLimitZeroDisablesThrottling(t *testing.T) {
	h := rateLimitedRouter(0)

	for i := 0; i < 50; i++ {
		if code := call(t, h, http.MethodPost, "/write", "203.0.113.10"); code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201 when the limit is disabled", i, code)
		}
	}
}

func TestRateLimitReturnsJSONError(t *testing.T) {
	h := rateLimitedRouter(1)
	call(t, h, http.MethodPost, "/write", "203.0.113.10")

	req := httptest.NewRequest(http.MethodPost, "/write", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if body := rec.Body.String(); !contains(body, `"success":false`) {
		t.Errorf("body = %s, want the standard error envelope", body)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
