package api

import (
	"bytes"
	"context"
	"encoding/json"
	imagepkg "image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"lostfound/internal/config"
	"lostfound/internal/database"
	"lostfound/internal/image"
)

// memStore is a stateful in-memory Store used to smoke-test the full HTTP
// stack (router, middleware, auth, handlers, image processing) end to end
// without a database. The SQL itself is covered by the repository
// integration tests in internal/database.
type memStore struct {
	mu           sync.Mutex
	users        map[uuid.UUID]*database.User
	posts        map[uuid.UUID]*database.Post
	interactions map[uuid.UUID]*database.Interaction
	reports      map[uuid.UUID]*database.Report
	alerts       map[uuid.UUID]*database.Alert
	adminID      uuid.UUID
}

func newMemStore() *memStore {
	admin := &database.User{
		ID:       uuid.New(),
		SSOID:    "admin_sso",
		Email:    "admin@college.edu",
		Name:     "Admin",
		Role:     "admin",
		IsActive: true,
	}
	return &memStore{
		users:        map[uuid.UUID]*database.User{admin.ID: admin},
		posts:        map[uuid.UUID]*database.Post{},
		interactions: map[uuid.UUID]*database.Interaction{},
		reports:      make(map[uuid.UUID]*database.Report),
		alerts:       make(map[uuid.UUID]*database.Alert),
		adminID:      admin.ID,
	}
}

func (m *memStore) GetBuildings(ctx context.Context) ([]database.Building, error) {
	return []database.Building{}, nil
}
func (m *memStore) GetBuildingByID(ctx context.Context, id uuid.UUID) (*database.Building, error) {
	return nil, nil
}
func (m *memStore) CreateBuilding(ctx context.Context, req database.CreateBuildingRequest) (*database.Building, error) {
	return &database.Building{ID: uuid.New(), Name: req.Name}, nil
}
func (m *memStore) GetLostFoundAreas(ctx context.Context) ([]database.LostFoundArea, error) {
	return []database.LostFoundArea{}, nil
}
func (m *memStore) GetLostFoundAreasByBuilding(ctx context.Context, buildingID uuid.UUID) ([]database.LostFoundArea, error) {
	return []database.LostFoundArea{}, nil
}
func (m *memStore) CreateLostFoundArea(ctx context.Context, req database.CreateLostFoundAreaRequest) (*database.LostFoundArea, error) {
	return &database.LostFoundArea{ID: uuid.New(), Name: req.Name}, nil
}

func (m *memStore) GetOrCreateUser(ctx context.Context, ssoUser database.SSOUser) (*database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.SSOID == ssoUser.SSOID {
			return u, nil
		}
	}
	user := &database.User{
		ID:       uuid.New(),
		SSOID:    ssoUser.SSOID,
		Email:    ssoUser.Email,
		Name:     ssoUser.Name,
		Role:     "user",
		IsActive: true,
	}
	m.users[user.ID] = user
	return user, nil
}

func (m *memStore) GetUserByID(ctx context.Context, id uuid.UUID) (*database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.users[id], nil
}

func (m *memStore) GetDefaultUserID(ctx context.Context) (uuid.UUID, error) {
	return m.adminID, nil
}

func (m *memStore) CreatePost(ctx context.Context, req database.CreatePostRequest, userID uuid.UUID, imageURLs []string) (*database.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	post := &database.Post{
		ID:          uuid.New(),
		Type:        req.Type,
		Category:    req.Category,
		Title:       req.Title,
		Description: req.Description,
		Location:    database.Point{Latitude: req.Latitude, Longitude: req.Longitude},
		PostedBy:    userID,
		IsLostItem:  req.IsLostItem,
		Status:      "active",
		EditToken:   uuid.NewString(),
		ImageURLs:   imageURLs,
		ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.posts[post.ID] = post
	return post, nil
}

// readCopy returns a copy of the post with the edit token stripped, matching
// the repository's behavior of never selecting edit_token on reads.
func readCopy(p *database.Post) *database.Post {
	cp := *p
	cp.EditToken = ""
	return &cp
}

func (m *memStore) SearchPosts(ctx context.Context, req database.SearchPostsRequest) (*database.SearchPostsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := &database.SearchPostsResponse{Limit: req.Limit, Offset: req.Offset}
	for _, p := range m.posts {
		if p.Status != "active" {
			continue
		}
		if req.Type != "" && p.Type != req.Type {
			continue
		}
		if req.Category != "" && p.Category != req.Category {
			continue
		}
		res.Posts = append(res.Posts, *readCopy(p))
	}
	res.Total = len(res.Posts)
	return res, nil
}

func (m *memStore) GetPostByID(ctx context.Context, id uuid.UUID) (*database.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.posts[id]
	if !ok {
		return nil, nil
	}
	return readCopy(p), nil
}

func (m *memStore) ClaimPost(ctx context.Context, postID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.posts[postID]
	if !ok || p.Status != "active" {
		return database.ErrNotFound
	}
	p.Status = "claimed"
	now := time.Now()
	p.ClaimedBy = &userID
	p.ClaimedAt = &now
	return nil
}

func (m *memStore) checkToken(id uuid.UUID, editToken string) (*database.Post, error) {
	p, ok := m.posts[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	if p.EditToken != editToken {
		return nil, database.ErrInvalidEditToken
	}
	return p, nil
}

func (m *memStore) UpdatePost(ctx context.Context, id uuid.UUID, editToken string, req database.UpdatePostRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, err := m.checkToken(id, editToken)
	if err != nil {
		return err
	}
	if req.Title != "" {
		p.Title = req.Title
	}
	if req.Description != "" {
		p.Description = req.Description
	}
	if req.Status != "" {
		p.Status = req.Status
	}
	return nil
}

func (m *memStore) DeletePost(ctx context.Context, id uuid.UUID, editToken string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, err := m.checkToken(id, editToken)
	if err != nil {
		return nil, err
	}
	delete(m.posts, id)
	return p.ImageURLs, nil
}

func (m *memStore) CreateInteraction(ctx context.Context, postID uuid.UUID, req database.CreateInteractionRequest) (*database.Interaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.posts[postID]
	if !ok {
		return nil, database.ErrNotFound
	}
	if p.Status != "active" {
		return nil, database.ErrPostNotActive
	}
	interaction := &database.Interaction{
		ID:              uuid.New(),
		PostID:          postID,
		InteractionType: req.InteractionType,
		ContactEmail:    req.ContactEmail,
		ContactName:     req.ContactName,
		Message:         req.Message,
		CreatedAt:       time.Now(),
		Status:          "pending",
	}
	m.interactions[interaction.ID] = interaction
	return interaction, nil
}

func (m *memStore) GetInteractionsByPost(ctx context.Context, postID uuid.UUID, editToken string) ([]database.Interaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.checkToken(postID, editToken); err != nil {
		return nil, err
	}
	var out []database.Interaction
	for _, i := range m.interactions {
		if i.PostID == postID {
			out = append(out, *i)
		}
	}
	return out, nil
}

func (m *memStore) UpdateInteractionStatus(ctx context.Context, interactionID uuid.UUID, editToken, newStatus string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i, ok := m.interactions[interactionID]
	if !ok {
		return database.ErrNotFound
	}
	p, err := m.checkToken(i.PostID, editToken)
	if err != nil {
		return err
	}
	i.Status = newStatus
	if newStatus == "accepted" && i.InteractionType == "claim" && p.Status == "active" {
		p.Status = "claimed"
	}
	return nil
}

// newSmokeServer assembles the router exactly like cmd/server does.
func newSmokeServer(t *testing.T) (*httptest.Server, *memStore, string) {
	t.Helper()

	store := newMemStore()
	uploadDir := t.TempDir()
	imgProcessor := image.NewProcessor(uploadDir, 10*1024*1024, []string{".jpg", ".jpeg", ".png", ".gif"})

	cfg := &config.Config{}
	cfg.Server.Environment = "development"
	cfg.JWT.Secret = "smoke-test-secret"
	cfg.JWT.Expiration = time.Hour

	handlers := NewHandler(store, imgProcessor)
	authHandlers := NewAuthHandler(store, cfg)

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Use(Middleware(cfg.JWT.Secret))
		r.Route("/auth", func(r chi.Router) {
			r.Post("/google", authHandlers.GoogleLogin)
			r.Post("/dev-login", authHandlers.DevLogin)
		})
		r.Route("/buildings", func(r chi.Router) {
			r.Get("/", handlers.GetBuildings)
			r.With(RequireAdmin).Post("/", handlers.CreateBuilding)
		})
		r.Route("/posts", func(r chi.Router) {
			r.Get("/", handlers.SearchPosts)
			r.Post("/", handlers.CreatePost)
			r.Get("/{id}", handlers.GetPostByID)
			r.Put("/{id}", handlers.UpdatePost)
			r.Delete("/{id}", handlers.DeletePost)
			r.Post("/{id}/claim", handlers.ClaimPost)
			r.Post("/{id}/interactions", handlers.CreateInteraction)
			r.Get("/{id}/interactions", handlers.GetPostInteractions)
			r.Post("/{id}/reports", handlers.CreateReport)
		})
		r.Put("/interactions/{id}", handlers.UpdateInteraction)
		r.Route("/reports", func(r chi.Router) {
			r.With(RequireAdmin).Get("/", handlers.GetReports)
			r.With(RequireAdmin).Put("/{id}", handlers.UpdateReport)
		})
		r.Route("/alerts", func(r chi.Router) {
			r.Post("/", handlers.CreateAlert)
			r.Get("/", handlers.GetAlerts)
			r.Delete("/{id}", handlers.DeleteAlert)
		})
	})

	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	return server, store, uploadDir
}

type apiEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func decodeEnvelope(t *testing.T, resp *http.Response, wantStatus int) apiEnvelope {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, wantStatus, buf.String())
	}
	var env apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return env
}

// TestFullUserJourney walks the complete product flow over real HTTP:
// sign in, post a found item with a photo, search, view, claim with a
// message, review and accept the claim, and clean up — verifying edit-token
// enforcement and image lifecycle along the way.
func TestFullUserJourney(t *testing.T) {
	server, _, uploadDir := newSmokeServer(t)
	client := server.Client()

	// 1. Dev sign-in issues a session token
	resp, err := client.Post(server.URL+"/api/auth/dev-login", "application/json",
		strings.NewReader(`{"email":"finder@college.edu","name":"Finder"}`))
	if err != nil {
		t.Fatalf("dev-login request failed: %v", err)
	}
	env := decodeEnvelope(t, resp, http.StatusOK)
	var login struct {
		Token string        `json:"token"`
		User  database.User `json:"user"`
	}
	if err := json.Unmarshal(env.Data, &login); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if login.Token == "" {
		t.Fatal("dev-login returned no token")
	}

	// 2. Create a found-item post with a real PNG attached
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for k, v := range map[string]string{
		"type": "found", "category": "item", "title": "Black water bottle",
		"description": "Found near the gym entrance",
		"latitude":    "40.7306", "longitude": "-73.9352",
	} {
		writer.WriteField(k, v)
	}
	part, _ := writer.CreateFormFile("images", "bottle.png")
	img := imagepkg.NewRGBA(imagepkg.Rect(0, 0, 400, 300))
	for x := 0; x < 400; x++ {
		for y := 0; y < 300; y++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	if err := png.Encode(part, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/posts", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+login.Token)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("create post failed: %v", err)
	}
	env = decodeEnvelope(t, resp, http.StatusCreated)
	var post database.Post
	if err := json.Unmarshal(env.Data, &post); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	if post.EditToken == "" {
		t.Fatal("creation response must include the edit token")
	}
	if post.PostedBy != login.User.ID {
		t.Errorf("post attributed to %v, want signed-in user %v", post.PostedBy, login.User.ID)
	}
	if len(post.ImageURLs) != 1 {
		t.Fatalf("image_urls = %v, want 1 entry", post.ImageURLs)
	}
	savedImage := filepath.Join(uploadDir, strings.TrimPrefix(post.ImageURLs[0], "/uploads/"))
	if _, err := os.Stat(savedImage); err != nil {
		t.Errorf("uploaded image not on disk: %v", err)
	}

	// 3. Search finds the post and never leaks the edit token
	resp, err = client.Get(server.URL + "/api/posts?type=found&category=item")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	raw := new(bytes.Buffer)
	raw.ReadFrom(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search status = %d; body: %s", resp.StatusCode, raw.String())
	}
	if !strings.Contains(raw.String(), post.ID.String()) {
		t.Error("search results do not contain the new post")
	}
	if strings.Contains(raw.String(), post.EditToken) || strings.Contains(raw.String(), "edit_token") {
		t.Error("search results leak the edit token")
	}

	// 4. The owner cannot be impersonated: update without token is rejected
	updateBody := `{"title":"hacked"}`
	req, _ = http.NewRequest(http.MethodPut, server.URL+"/api/posts/"+post.ID.String(), strings.NewReader(updateBody))
	resp, _ = client.Do(req)
	decodeEnvelopeError(t, resp, http.StatusUnauthorized)

	req, _ = http.NewRequest(http.MethodPut, server.URL+"/api/posts/"+post.ID.String(), strings.NewReader(updateBody))
	req.Header.Set("X-Edit-Token", "wrong-token")
	resp, _ = client.Do(req)
	decodeEnvelopeError(t, resp, http.StatusForbidden)

	// 5. A student claims the item with a message
	claimBody := `{"interaction_type":"claim","contact_email":"owner@college.edu","contact_name":"Owner","message":"It has a dent on the cap"}`
	resp, err = client.Post(server.URL+"/api/posts/"+post.ID.String()+"/interactions", "application/json", strings.NewReader(claimBody))
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	env = decodeEnvelope(t, resp, http.StatusCreated)
	var interaction database.Interaction
	if err := json.Unmarshal(env.Data, &interaction); err != nil {
		t.Fatalf("decode interaction: %v", err)
	}

	// 6. Only the poster can read the claims inbox
	resp, _ = client.Get(server.URL + "/api/posts/" + post.ID.String() + "/interactions")
	decodeEnvelopeError(t, resp, http.StatusUnauthorized)

	req, _ = http.NewRequest(http.MethodGet, server.URL+"/api/posts/"+post.ID.String()+"/interactions", nil)
	req.Header.Set("X-Edit-Token", post.EditToken)
	resp, _ = client.Do(req)
	env = decodeEnvelope(t, resp, http.StatusOK)
	if !strings.Contains(string(env.Data), "owner@college.edu") {
		t.Error("claims inbox is missing the claimer's contact info")
	}

	// 7. Accepting the claim marks the post claimed
	req, _ = http.NewRequest(http.MethodPut, server.URL+"/api/interactions/"+interaction.ID.String(),
		strings.NewReader(`{"status":"accepted"}`))
	req.Header.Set("X-Edit-Token", post.EditToken)
	resp, _ = client.Do(req)
	decodeEnvelope(t, resp, http.StatusOK)

	resp, _ = client.Get(server.URL + "/api/posts/" + post.ID.String())
	env = decodeEnvelope(t, resp, http.StatusOK)
	var claimed database.Post
	json.Unmarshal(env.Data, &claimed)
	if claimed.Status != "claimed" {
		t.Errorf("post status = %q, want claimed", claimed.Status)
	}

	// 8. Deleting with the token removes the post and its image file
	req, _ = http.NewRequest(http.MethodDelete, server.URL+"/api/posts/"+post.ID.String(), nil)
	req.Header.Set("X-Edit-Token", post.EditToken)
	resp, _ = client.Do(req)
	decodeEnvelope(t, resp, http.StatusOK)

	if _, err := os.Stat(savedImage); !os.IsNotExist(err) {
		t.Error("image file still on disk after post deletion")
	}
	resp, _ = client.Get(server.URL + "/api/posts/" + post.ID.String())
	decodeEnvelopeError(t, resp, http.StatusNotFound)
}

// TestAdminGating verifies role enforcement over real HTTP.
func TestAdminGating(t *testing.T) {
	server, store, _ := newSmokeServer(t)
	client := server.Client()

	buildingBody := func() *strings.Reader {
		return strings.NewReader(`{"name":"New Hall","latitude":40.73,"longitude":-73.93}`)
	}

	// Anonymous: forbidden
	resp, _ := client.Post(server.URL+"/api/buildings", "application/json", buildingBody())
	decodeEnvelopeError(t, resp, http.StatusForbidden)

	// Regular user: forbidden
	resp, _ = client.Post(server.URL+"/api/auth/dev-login", "application/json",
		strings.NewReader(`{"email":"student@college.edu"}`))
	env := decodeEnvelope(t, resp, http.StatusOK)
	var login struct {
		Token string `json:"token"`
	}
	json.Unmarshal(env.Data, &login)

	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/buildings", buildingBody())
	req.Header.Set("Authorization", "Bearer "+login.Token)
	resp, _ = client.Do(req)
	decodeEnvelopeError(t, resp, http.StatusForbidden)

	// Admin: allowed. Promote via the store, sign in again.
	store.mu.Lock()
	for _, u := range store.users {
		if u.Email == "student@college.edu" {
			u.Role = "admin"
		}
	}
	store.mu.Unlock()

	resp, _ = client.Post(server.URL+"/api/auth/dev-login", "application/json",
		strings.NewReader(`{"email":"student@college.edu"}`))
	env = decodeEnvelope(t, resp, http.StatusOK)
	json.Unmarshal(env.Data, &login)

	req, _ = http.NewRequest(http.MethodPost, server.URL+"/api/buildings", buildingBody())
	req.Header.Set("Authorization", "Bearer "+login.Token)
	resp, _ = client.Do(req)
	decodeEnvelope(t, resp, http.StatusCreated)

	// Garbage bearer token: rejected outright
	req, _ = http.NewRequest(http.MethodPost, server.URL+"/api/buildings", buildingBody())
	req.Header.Set("Authorization", "Bearer nonsense")
	resp, _ = client.Do(req)
	decodeEnvelopeError(t, resp, http.StatusUnauthorized)
}

func decodeEnvelopeError(t *testing.T, resp *http.Response, wantStatus int) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, wantStatus, buf.String())
	}
	var env apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if env.Success {
		t.Errorf("expected success=false for status %d", wantStatus)
	}
}

// Reports and alerts: a real in-memory implementation, so the smoke tests
// exercise the handlers end to end over HTTP rather than asserting on stubs.

func (m *memStore) CreateReport(ctx context.Context, postID uuid.UUID, req database.CreateReportRequest) (*database.Report, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.posts[postID]; !ok {
		return nil, database.ErrNotFound
	}
	rep := &database.Report{
		ID:            uuid.New(),
		PostID:        &postID,
		ReporterEmail: req.ReporterEmail,
		Reason:        req.Reason,
		Description:   req.Description,
		CreatedAt:     time.Now(),
		Status:        "pending",
	}
	m.reports[rep.ID] = rep
	return rep, nil
}

func (m *memStore) GetReports(ctx context.Context, status string) ([]database.Report, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []database.Report{}
	for _, r := range m.reports {
		if status == "" || r.Status == status {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (m *memStore) UpdateReportStatus(ctx context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reports[id]
	if !ok {
		return database.ErrNotFound
	}
	r.Status = status
	return nil
}

func (m *memStore) CreateAlert(ctx context.Context, req database.CreateAlertRequest) (*database.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := &database.Alert{
		ID:           uuid.New(),
		Email:        req.Email,
		Location:     database.Point{Latitude: req.Latitude, Longitude: req.Longitude},
		RadiusMeters: req.RadiusMeters,
		Categories:   req.Categories,
		Keywords:     req.Keywords,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}
	m.alerts[a.ID] = a
	return a, nil
}

func (m *memStore) GetAlertsByEmail(ctx context.Context, email string) ([]database.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []database.Alert{}
	for _, a := range m.alerts {
		if a.Email == email && a.IsActive {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m *memStore) DeactivateAlert(ctx context.Context, id uuid.UUID, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.alerts[id]
	if !ok || a.Email != email || !a.IsActive {
		return database.ErrNotFound
	}
	a.IsActive = false
	return nil
}

// TestReportingFlow covers filing a report as an anonymous visitor and
// triaging it as a moderator, including the gate that keeps reports away from
// ordinary users.
func TestReportingFlow(t *testing.T) {
	server, store, _ := newSmokeServer(t)
	client := server.Client()

	// A post to report.
	postID := uuid.New()
	store.mu.Lock()
	store.posts[postID] = &database.Post{
		ID: postID, Type: "found", Category: "item", Title: "Suspicious listing",
		Status: "active", EditToken: "tok", PostedBy: store.adminID,
	}
	store.mu.Unlock()

	// Anyone can file a report -- no sign-in required.
	resp, err := client.Post(server.URL+"/api/posts/"+postID.String()+"/reports", "application/json",
		strings.NewReader(`{"reason":"spam","description":"Posted 40 times","reporter_email":"witness@college.edu"}`))
	if err != nil {
		t.Fatalf("file report: %v", err)
	}
	env := decodeEnvelope(t, resp, http.StatusCreated)
	var report database.Report
	if err := json.Unmarshal(env.Data, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Status != "pending" {
		t.Errorf("status = %q, want pending", report.Status)
	}

	// An unknown reason is rejected.
	resp, _ = client.Post(server.URL+"/api/posts/"+postID.String()+"/reports", "application/json",
		strings.NewReader(`{"reason":"because"}`))
	decodeEnvelopeError(t, resp, http.StatusBadRequest)

	// Reporting a post that does not exist is a 404, not a dangling row.
	resp, _ = client.Post(server.URL+"/api/posts/"+uuid.New().String()+"/reports", "application/json",
		strings.NewReader(`{"reason":"spam"}`))
	decodeEnvelopeError(t, resp, http.StatusNotFound)

	// Reading reports is admin-only: anonymous and ordinary users are refused.
	resp, _ = client.Get(server.URL + "/api/reports")
	decodeEnvelopeError(t, resp, http.StatusForbidden)

	resp, _ = client.Post(server.URL+"/api/auth/dev-login", "application/json",
		strings.NewReader(`{"email":"student@college.edu"}`))
	env = decodeEnvelope(t, resp, http.StatusOK)
	var login struct {
		Token string `json:"token"`
	}
	json.Unmarshal(env.Data, &login)

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/reports", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	resp, _ = client.Do(req)
	decodeEnvelopeError(t, resp, http.StatusForbidden)

	// Promote and sign in again: now the report is visible and can be resolved.
	store.mu.Lock()
	for _, u := range store.users {
		if u.Email == "student@college.edu" {
			u.Role = "admin"
		}
	}
	store.mu.Unlock()

	resp, _ = client.Post(server.URL+"/api/auth/dev-login", "application/json",
		strings.NewReader(`{"email":"student@college.edu"}`))
	env = decodeEnvelope(t, resp, http.StatusOK)
	json.Unmarshal(env.Data, &login)

	req, _ = http.NewRequest(http.MethodGet, server.URL+"/api/reports?status=pending", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	resp, _ = client.Do(req)
	env = decodeEnvelope(t, resp, http.StatusOK)
	var list database.ReportsResponse
	json.Unmarshal(env.Data, &list)
	if list.Total != 1 {
		t.Fatalf("total = %d, want 1", list.Total)
	}

	req, _ = http.NewRequest(http.MethodPut, server.URL+"/api/reports/"+report.ID.String(),
		strings.NewReader(`{"status":"resolved"}`))
	req.Header.Set("Authorization", "Bearer "+login.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	decodeEnvelope(t, resp, http.StatusOK)

	store.mu.Lock()
	got := store.reports[report.ID].Status
	store.mu.Unlock()
	if got != "resolved" {
		t.Errorf("stored status = %q, want resolved", got)
	}
}

// TestAlertSubscriptionFlow covers subscribing to nearby posts, listing your
// own subscriptions, and unsubscribing -- including that an alert ID alone is
// not enough to cancel someone else's subscription.
func TestAlertSubscriptionFlow(t *testing.T) {
	server, _, _ := newSmokeServer(t)
	client := server.Client()

	resp, err := client.Post(server.URL+"/api/alerts", "application/json",
		strings.NewReader(`{"email":"Watcher@College.edu","latitude":40.73,"longitude":-73.93,"radius_meters":2000,"categories":["item","document"]}`))
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	env := decodeEnvelope(t, resp, http.StatusCreated)
	var alert database.Alert
	json.Unmarshal(env.Data, &alert)
	if alert.Email != "watcher@college.edu" {
		t.Errorf("email = %q, want it normalised to lowercase", alert.Email)
	}
	if !alert.IsActive {
		t.Error("a new alert should be active")
	}

	// Validation.
	for _, body := range []string{
		`{"email":"nope","latitude":40.73,"longitude":-73.93}`,
		`{"email":"a@b.com","latitude":91,"longitude":-73.93}`,
		`{"email":"a@b.com","latitude":40.73,"longitude":-73.93,"radius_meters":999999}`,
		`{"email":"a@b.com","latitude":40.73,"longitude":-73.93,"categories":["nonsense"]}`,
	} {
		resp, _ = client.Post(server.URL+"/api/alerts", "application/json", strings.NewReader(body))
		decodeEnvelopeError(t, resp, http.StatusBadRequest)
	}

	// Listing requires an email and returns only that address's alerts.
	resp, _ = client.Get(server.URL + "/api/alerts")
	decodeEnvelopeError(t, resp, http.StatusBadRequest)

	resp, _ = client.Get(server.URL + "/api/alerts?email=watcher@college.edu")
	env = decodeEnvelope(t, resp, http.StatusOK)
	var list database.AlertsResponse
	json.Unmarshal(env.Data, &list)
	if list.Total != 1 {
		t.Fatalf("total = %d, want 1", list.Total)
	}

	resp, _ = client.Get(server.URL + "/api/alerts?email=someone.else@college.edu")
	env = decodeEnvelope(t, resp, http.StatusOK)
	json.Unmarshal(env.Data, &list)
	if list.Total != 0 {
		t.Errorf("total = %d, want 0 for a different address", list.Total)
	}

	// Someone else's email cannot cancel this alert.
	req, _ := http.NewRequest(http.MethodDelete,
		server.URL+"/api/alerts/"+alert.ID.String()+"?email=attacker@college.edu", nil)
	resp, _ = client.Do(req)
	decodeEnvelopeError(t, resp, http.StatusNotFound)

	// The owner can.
	req, _ = http.NewRequest(http.MethodDelete,
		server.URL+"/api/alerts/"+alert.ID.String()+"?email=watcher@college.edu", nil)
	resp, _ = client.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()

	resp, _ = client.Get(server.URL + "/api/alerts?email=watcher@college.edu")
	env = decodeEnvelope(t, resp, http.StatusOK)
	json.Unmarshal(env.Data, &list)
	if list.Total != 0 {
		t.Errorf("total = %d, want 0 after unsubscribing", list.Total)
	}
}
