package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newSPARouter(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	must := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	must("index.html", "<!doctype html><div id=root></div>")
	if err := os.MkdirAll(filepath.Join(dir, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	must(filepath.Join("static", "app.js"), "console.log(1)")

	r := chi.NewRouter()
	r.Get("/api/posts", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("[]")) })
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("OK")) })
	mountSPA(r, dir)
	return r, dir
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSPAServesIndexForClientRoutes(t *testing.T) {
	h, _ := newSPARouter(t)
	// A deep link must return the shell so a page reload does not 404.
	for _, p := range []string{"/", "/posts/abc-123", "/create"} {
		rec := get(t, h, p)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", p, rec.Code)
		}
		if !contains(rec.Body.String(), "id=root") {
			t.Errorf("%s: body is not index.html", p)
		}
	}
}

func TestSPAServesRealAssets(t *testing.T) {
	h, _ := newSPARouter(t)
	rec := get(t, h, "/static/app.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !contains(rec.Body.String(), "console.log") {
		t.Error("asset was replaced by the SPA shell")
	}
}

func TestSPADoesNotSwallowAPIRoutes(t *testing.T) {
	h, _ := newSPARouter(t)

	// Real API routes still work.
	if rec := get(t, h, "/api/posts"); rec.Code != http.StatusOK {
		t.Errorf("/api/posts: status = %d, want 200", rec.Code)
	}
	if rec := get(t, h, "/health"); rec.Code != http.StatusOK {
		t.Errorf("/health: status = %d, want 200", rec.Code)
	}

	// Unknown API and upload paths must 404 rather than return HTML: a client
	// expecting JSON should see a 404, not a page it cannot parse.
	for _, p := range []string{"/api/nope", "/uploads/missing.png"} {
		rec := get(t, h, p)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", p, rec.Code)
		}
		if contains(rec.Body.String(), "id=root") {
			t.Errorf("%s: returned the SPA shell instead of a 404", p)
		}
	}
}

func TestSPAIgnoredWhenBuildMissing(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/posts", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("[]")) })
	mountSPA(r, t.TempDir()) // no index.html

	// Without a build, unknown paths keep chi's default 404.
	if rec := get(t, r, "/posts/abc"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when no build is present", rec.Code)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
