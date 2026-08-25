package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"lostfound/internal/auth"
	"lostfound/internal/database"
	"lostfound/internal/image"
)

// mockStore implements Store with overridable function fields.
type mockStore struct {
	getBuildings            func(ctx context.Context) ([]database.Building, error)
	getBuildingByID         func(ctx context.Context, id uuid.UUID) (*database.Building, error)
	createBuilding          func(ctx context.Context, req database.CreateBuildingRequest) (*database.Building, error)
	getAreas                func(ctx context.Context) ([]database.LostFoundArea, error)
	getAreasByBuilding      func(ctx context.Context, buildingID uuid.UUID) ([]database.LostFoundArea, error)
	createArea              func(ctx context.Context, req database.CreateLostFoundAreaRequest) (*database.LostFoundArea, error)
	getOrCreateUser         func(ctx context.Context, ssoUser database.SSOUser) (*database.User, error)
	getUserByID             func(ctx context.Context, id uuid.UUID) (*database.User, error)
	getDefaultUserID        func(ctx context.Context) (uuid.UUID, error)
	createPost              func(ctx context.Context, req database.CreatePostRequest, userID uuid.UUID, imageURLs []string) (*database.Post, error)
	searchPosts             func(ctx context.Context, req database.SearchPostsRequest) (*database.SearchPostsResponse, error)
	getPostByID             func(ctx context.Context, id uuid.UUID) (*database.Post, error)
	claimPost               func(ctx context.Context, postID, userID uuid.UUID) error
	updatePost              func(ctx context.Context, id uuid.UUID, editToken string, req database.UpdatePostRequest) error
	deletePost              func(ctx context.Context, id uuid.UUID, editToken string) ([]string, error)
	createInteraction       func(ctx context.Context, postID uuid.UUID, req database.CreateInteractionRequest) (*database.Interaction, error)
	getInteractionsByPost   func(ctx context.Context, postID uuid.UUID, editToken string) ([]database.Interaction, error)
	updateInteractionStatus func(ctx context.Context, interactionID uuid.UUID, editToken, newStatus string) error
}

func (m *mockStore) GetBuildings(ctx context.Context) ([]database.Building, error) {
	return m.getBuildings(ctx)
}
func (m *mockStore) GetBuildingByID(ctx context.Context, id uuid.UUID) (*database.Building, error) {
	return m.getBuildingByID(ctx, id)
}
func (m *mockStore) CreateBuilding(ctx context.Context, req database.CreateBuildingRequest) (*database.Building, error) {
	return m.createBuilding(ctx, req)
}
func (m *mockStore) GetLostFoundAreas(ctx context.Context) ([]database.LostFoundArea, error) {
	return m.getAreas(ctx)
}
func (m *mockStore) GetLostFoundAreasByBuilding(ctx context.Context, buildingID uuid.UUID) ([]database.LostFoundArea, error) {
	return m.getAreasByBuilding(ctx, buildingID)
}
func (m *mockStore) CreateLostFoundArea(ctx context.Context, req database.CreateLostFoundAreaRequest) (*database.LostFoundArea, error) {
	return m.createArea(ctx, req)
}
func (m *mockStore) GetOrCreateUser(ctx context.Context, ssoUser database.SSOUser) (*database.User, error) {
	return m.getOrCreateUser(ctx, ssoUser)
}
func (m *mockStore) GetUserByID(ctx context.Context, id uuid.UUID) (*database.User, error) {
	return m.getUserByID(ctx, id)
}
func (m *mockStore) GetDefaultUserID(ctx context.Context) (uuid.UUID, error) {
	return m.getDefaultUserID(ctx)
}
func (m *mockStore) CreatePost(ctx context.Context, req database.CreatePostRequest, userID uuid.UUID, imageURLs []string) (*database.Post, error) {
	return m.createPost(ctx, req, userID, imageURLs)
}
func (m *mockStore) SearchPosts(ctx context.Context, req database.SearchPostsRequest) (*database.SearchPostsResponse, error) {
	return m.searchPosts(ctx, req)
}
func (m *mockStore) GetPostByID(ctx context.Context, id uuid.UUID) (*database.Post, error) {
	return m.getPostByID(ctx, id)
}
func (m *mockStore) ClaimPost(ctx context.Context, postID, userID uuid.UUID) error {
	return m.claimPost(ctx, postID, userID)
}
func (m *mockStore) UpdatePost(ctx context.Context, id uuid.UUID, editToken string, req database.UpdatePostRequest) error {
	return m.updatePost(ctx, id, editToken, req)
}
func (m *mockStore) DeletePost(ctx context.Context, id uuid.UUID, editToken string) ([]string, error) {
	return m.deletePost(ctx, id, editToken)
}
func (m *mockStore) CreateInteraction(ctx context.Context, postID uuid.UUID, req database.CreateInteractionRequest) (*database.Interaction, error) {
	return m.createInteraction(ctx, postID, req)
}
func (m *mockStore) GetInteractionsByPost(ctx context.Context, postID uuid.UUID, editToken string) ([]database.Interaction, error) {
	return m.getInteractionsByPost(ctx, postID, editToken)
}
func (m *mockStore) UpdateInteractionStatus(ctx context.Context, interactionID uuid.UUID, editToken, newStatus string) error {
	return m.updateInteractionStatus(ctx, interactionID, editToken, newStatus)
}

// newTestRouter mounts the handlers on the same routes as cmd/server.
func newTestRouter(t *testing.T, store *mockStore) *chi.Mux {
	t.Helper()
	h := NewHandler(store, image.NewProcessor(t.TempDir(), 10*1024*1024, []string{".jpg", ".png"}))
	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Route("/posts", func(r chi.Router) {
			r.Get("/", h.SearchPosts)
			r.Post("/", h.CreatePost)
			r.Get("/{id}", h.GetPostByID)
			r.Put("/{id}", h.UpdatePost)
			r.Delete("/{id}", h.DeletePost)
			r.Post("/{id}/claim", h.ClaimPost)
			r.Post("/{id}/interactions", h.CreateInteraction)
			r.Get("/{id}/interactions", h.GetPostInteractions)
		})
		r.Put("/interactions/{id}", h.UpdateInteraction)
	})
	return r
}

func doJSON(t *testing.T, router http.Handler, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestUpdatePostRequiresEditToken(t *testing.T) {
	router := newTestRouter(t, &mockStore{})
	rec := doJSON(t, router, http.MethodPut, "/api/posts/"+uuid.NewString(), map[string]string{"title": "x"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestUpdatePostInvalidToken(t *testing.T) {
	store := &mockStore{
		updatePost: func(ctx context.Context, id uuid.UUID, editToken string, req database.UpdatePostRequest) error {
			return database.ErrInvalidEditToken
		},
	}
	router := newTestRouter(t, store)
	rec := doJSON(t, router, http.MethodPut, "/api/posts/"+uuid.NewString(),
		map[string]string{"title": "x"}, map[string]string{"X-Edit-Token": "wrong"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestUpdatePostNotFound(t *testing.T) {
	store := &mockStore{
		updatePost: func(ctx context.Context, id uuid.UUID, editToken string, req database.UpdatePostRequest) error {
			return database.ErrNotFound
		},
	}
	router := newTestRouter(t, store)
	rec := doJSON(t, router, http.MethodPut, "/api/posts/"+uuid.NewString(),
		map[string]string{"title": "x"}, map[string]string{"X-Edit-Token": "token"})
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestUpdatePostRejectsInvalidStatus(t *testing.T) {
	router := newTestRouter(t, &mockStore{})
	rec := doJSON(t, router, http.MethodPut, "/api/posts/"+uuid.NewString(),
		map[string]string{"status": "banana"}, map[string]string{"X-Edit-Token": "token"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestDeletePostRequiresEditToken(t *testing.T) {
	router := newTestRouter(t, &mockStore{})
	rec := doJSON(t, router, http.MethodDelete, "/api/posts/"+uuid.NewString(), nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestClaimPostNotFound(t *testing.T) {
	store := &mockStore{
		getDefaultUserID: func(ctx context.Context) (uuid.UUID, error) { return uuid.New(), nil },
		claimPost: func(ctx context.Context, postID, userID uuid.UUID) error {
			return database.ErrNotFound
		},
	}
	router := newTestRouter(t, store)
	rec := doJSON(t, router, http.MethodPost, "/api/posts/"+uuid.NewString()+"/claim", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func postForm(t *testing.T, router http.Handler, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range fields {
		writer.WriteField(k, v)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/posts", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func validPostFields() map[string]string {
	return map[string]string{
		"type":      "found",
		"category":  "item",
		"title":     "Blue backpack",
		"latitude":  "40.7306",
		"longitude": "-73.9352",
	}
}

func TestCreatePostValidation(t *testing.T) {
	// The store's createPost must never be reached for invalid input
	store := &mockStore{
		getDefaultUserID: func(ctx context.Context) (uuid.UUID, error) { return uuid.New(), nil },
		createPost: func(ctx context.Context, req database.CreatePostRequest, userID uuid.UUID, imageURLs []string) (*database.Post, error) {
			t.Error("CreatePost should not be called for invalid input")
			return nil, nil
		},
	}
	router := newTestRouter(t, store)

	cases := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{"invalid type", func(f map[string]string) { f["type"] = "stolen" }},
		{"invalid category", func(f map[string]string) { f["category"] = "car" }},
		{"missing title", func(f map[string]string) { f["title"] = "  " }},
		{"bad latitude", func(f map[string]string) { f["latitude"] = "999" }},
		{"bad longitude", func(f map[string]string) { f["longitude"] = "not-a-number" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := validPostFields()
			tc.mutate(fields)
			rec := postForm(t, router, fields)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestCreatePostSuccessReturnsEditToken(t *testing.T) {
	userID := uuid.New()
	store := &mockStore{
		getDefaultUserID: func(ctx context.Context) (uuid.UUID, error) { return userID, nil },
		createPost: func(ctx context.Context, req database.CreatePostRequest, gotUserID uuid.UUID, imageURLs []string) (*database.Post, error) {
			if gotUserID != userID {
				t.Errorf("userID = %v, want %v", gotUserID, userID)
			}
			return &database.Post{
				ID:        uuid.New(),
				Type:      req.Type,
				Category:  req.Category,
				Title:     req.Title,
				EditToken: "secret-token",
				CreatedAt: time.Now(),
			}, nil
		},
	}
	router := newTestRouter(t, store)

	rec := postForm(t, router, validPostFields())
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if !strings.Contains(rec.Body.String(), "secret-token") {
		t.Error("creation response should include the edit token")
	}
}

func TestGetPostByIDOmitsEditToken(t *testing.T) {
	store := &mockStore{
		getPostByID: func(ctx context.Context, id uuid.UUID) (*database.Post, error) {
			// Repository never selects edit_token on reads; the JSON tag's
			// omitempty keeps an empty token out of the response entirely.
			return &database.Post{ID: id, Type: "found", Category: "item", Title: "Keys"}, nil
		},
	}
	router := newTestRouter(t, store)

	rec := doJSON(t, router, http.MethodGet, "/api/posts/"+uuid.NewString(), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "edit_token") {
		t.Error("read response must not contain edit_token")
	}
}

func TestSearchPostsParsesParams(t *testing.T) {
	var got database.SearchPostsRequest
	store := &mockStore{
		searchPosts: func(ctx context.Context, req database.SearchPostsRequest) (*database.SearchPostsResponse, error) {
			got = req
			return &database.SearchPostsResponse{Posts: []database.Post{}}, nil
		},
	}
	router := newTestRouter(t, store)

	rec := doJSON(t, router, http.MethodGet,
		"/api/posts?lat=40.5&lng=-73.9&radius=2000&type=lost&category=pet&is_lost_item=true&limit=10&offset=5", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if got.Latitude != 40.5 || got.Longitude != -73.9 {
		t.Errorf("location = (%v, %v), want (40.5, -73.9)", got.Latitude, got.Longitude)
	}
	if got.Radius != 2000 {
		t.Errorf("radius = %d, want 2000", got.Radius)
	}
	if got.Type != "lost" || got.Category != "pet" {
		t.Errorf("type/category = %q/%q, want lost/pet", got.Type, got.Category)
	}
	if got.IsLostItem == nil || !*got.IsLostItem {
		t.Error("is_lost_item should be parsed as true")
	}
	if got.Limit != 10 || got.Offset != 5 {
		t.Errorf("limit/offset = %d/%d, want 10/5", got.Limit, got.Offset)
	}
}

func TestCreateInteractionValidation(t *testing.T) {
	store := &mockStore{
		createInteraction: func(ctx context.Context, postID uuid.UUID, req database.CreateInteractionRequest) (*database.Interaction, error) {
			t.Error("CreateInteraction should not be called for invalid input")
			return nil, nil
		},
	}
	router := newTestRouter(t, store)
	postPath := "/api/posts/" + uuid.NewString() + "/interactions"

	rec := doJSON(t, router, http.MethodPost, postPath,
		map[string]string{"interaction_type": "steal", "contact_email": "a@b.edu"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid type: status = %d, want 400", rec.Code)
	}

	rec = doJSON(t, router, http.MethodPost, postPath,
		map[string]string{"interaction_type": "claim", "contact_email": "not-an-email"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid email: status = %d, want 400", rec.Code)
	}
}

func TestCreateInteractionOnInactivePost(t *testing.T) {
	store := &mockStore{
		createInteraction: func(ctx context.Context, postID uuid.UUID, req database.CreateInteractionRequest) (*database.Interaction, error) {
			return nil, database.ErrPostNotActive
		},
	}
	router := newTestRouter(t, store)

	rec := doJSON(t, router, http.MethodPost, "/api/posts/"+uuid.NewString()+"/interactions",
		map[string]string{"interaction_type": "claim", "contact_email": "a@b.edu", "message": "mine"}, nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	var gotUserID, gotRole string
	probe := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = r.Context().Value(UserIDKey).(string)
		gotRole, _ = r.Context().Value(RoleKey).(string)
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware(secret)(probe)

	// No token: anonymous but allowed through
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("anonymous: status = %d, want 200", rec.Code)
	}

	// Valid token: user in context
	token, err := auth.IssueToken(secret, auth.AppClaims{UserID: userID, Role: "admin"}, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid token: status = %d, want 200", rec.Code)
	}
	if gotUserID != userID.String() {
		t.Errorf("context user = %q, want %q", gotUserID, userID.String())
	}
	if gotRole != "admin" {
		t.Errorf("context role = %q, want admin", gotRole)
	}

	// Invalid token: rejected
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("invalid token: status = %d, want 401", rec.Code)
	}
}

func TestRequireAdmin(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := RequireAdmin(ok)

	// No role: forbidden
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("anonymous: status = %d, want 403", rec.Code)
	}

	// Non-admin role: forbidden
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(contextWithUser(req.Context(), uuid.NewString(), "user"))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("user role: status = %d, want 403", rec.Code)
	}

	// Admin role: allowed
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(contextWithUser(req.Context(), uuid.NewString(), "admin"))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("admin role: status = %d, want 200", rec.Code)
	}
}
