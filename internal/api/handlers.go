package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"lostfound/internal/database"
	"lostfound/internal/image"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

// UserIDKey is the context key under which auth middleware stores the user ID
const UserIDKey contextKey = "user_id"

// maxImagesPerPost caps how many images can be attached to a single post
const maxImagesPerPost = 5

var validTypes = map[string]bool{"lost": true, "found": true}
var validCategories = map[string]bool{"pet": true, "document": true, "item": true, "other": true}
var validStatuses = map[string]bool{"active": true, "claimed": true, "resolved": true}
var validInteractionTypes = map[string]bool{"claim": true, "help": true, "report": true}
var validReportReasons = map[string]bool{"spam": true, "inappropriate": true, "fraudulent": true, "wrong_info": true, "other": true}
var validReportStatuses = map[string]bool{"pending": true, "reviewed": true, "resolved": true}
var validInteractionStatuses = map[string]bool{"accepted": true, "rejected": true}

// Store is the data-access interface the HTTP handlers depend on. It is
// implemented by *database.Repository and by mocks in tests.
type Store interface {
	GetBuildings(ctx context.Context) ([]database.Building, error)
	GetBuildingByID(ctx context.Context, id uuid.UUID) (*database.Building, error)
	CreateBuilding(ctx context.Context, req database.CreateBuildingRequest) (*database.Building, error)
	GetLostFoundAreas(ctx context.Context) ([]database.LostFoundArea, error)
	GetLostFoundAreasByBuilding(ctx context.Context, buildingID uuid.UUID) ([]database.LostFoundArea, error)
	CreateLostFoundArea(ctx context.Context, req database.CreateLostFoundAreaRequest) (*database.LostFoundArea, error)
	GetOrCreateUser(ctx context.Context, ssoUser database.SSOUser) (*database.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*database.User, error)
	GetDefaultUserID(ctx context.Context) (uuid.UUID, error)
	CreatePost(ctx context.Context, req database.CreatePostRequest, userID uuid.UUID, imageURLs []string) (*database.Post, error)
	SearchPosts(ctx context.Context, req database.SearchPostsRequest) (*database.SearchPostsResponse, error)
	GetPostByID(ctx context.Context, id uuid.UUID) (*database.Post, error)
	ClaimPost(ctx context.Context, postID, userID uuid.UUID) error
	UpdatePost(ctx context.Context, id uuid.UUID, editToken string, req database.UpdatePostRequest) error
	DeletePost(ctx context.Context, id uuid.UUID, editToken string) ([]string, error)
	CreateInteraction(ctx context.Context, postID uuid.UUID, req database.CreateInteractionRequest) (*database.Interaction, error)
	GetInteractionsByPost(ctx context.Context, postID uuid.UUID, editToken string) ([]database.Interaction, error)
	UpdateInteractionStatus(ctx context.Context, interactionID uuid.UUID, editToken, newStatus string) error
	CreateReport(ctx context.Context, postID uuid.UUID, req database.CreateReportRequest) (*database.Report, error)
	GetReports(ctx context.Context, status string) ([]database.Report, error)
	UpdateReportStatus(ctx context.Context, id uuid.UUID, status string) error
	CreateAlert(ctx context.Context, req database.CreateAlertRequest) (*database.Alert, error)
	GetAlertsByEmail(ctx context.Context, email string) ([]database.Alert, error)
	DeactivateAlert(ctx context.Context, id uuid.UUID, email string) error
}

// Handler provides HTTP handlers for the API
type Handler struct {
	repo         Store
	imgProcessor *image.Processor

	defaultUserOnce sync.Once
	defaultUserID   uuid.UUID
	defaultUserErr  error
}

// NewHandler creates a new handler instance
func NewHandler(repo Store, imgProcessor *image.Processor) *Handler {
	return &Handler{
		repo:         repo,
		imgProcessor: imgProcessor,
	}
}

// writeJSON writes a JSON success response with the proper Content-Type
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(database.APIResponse{Success: true, Data: data}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// writeError writes a JSON error response. Internal details are logged, never
// sent to the client.
func writeError(w http.ResponseWriter, status int, publicMsg string, internal error) {
	if internal != nil {
		log.Printf("api error (%d %s): %v", status, publicMsg, internal)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(database.APIResponse{Success: false, Error: publicMsg}); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

// currentUserID resolves the acting user: the authenticated user from context
// if present, otherwise the default admin user looked up once from the
// database (a stopgap until real authentication is implemented).
func (h *Handler) currentUserID(r *http.Request) (uuid.UUID, error) {
	if v := r.Context().Value(UserIDKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				return uuid.Nil, errors.New("invalid user ID in context")
			}
			return id, nil
		}
		if id, ok := v.(uuid.UUID); ok {
			return id, nil
		}
	}

	h.defaultUserOnce.Do(func() {
		h.defaultUserID, h.defaultUserErr = h.repo.GetDefaultUserID(r.Context())
	})
	if h.defaultUserErr != nil {
		return uuid.Nil, h.defaultUserErr
	}
	return h.defaultUserID, nil
}

// editTokenFromRequest extracts the edit token from the X-Edit-Token header
// (preferred) or the edit_token query parameter.
func editTokenFromRequest(r *http.Request) string {
	if token := r.Header.Get("X-Edit-Token"); token != "" {
		return token
	}
	return r.URL.Query().Get("edit_token")
}

// stripUploadsPrefix converts a stored image URL like "/uploads/abc.jpg" to the
// bare filename the image processor works with.
func stripUploadsPrefix(url string) string {
	return strings.TrimPrefix(strings.TrimPrefix(url, "/uploads/"), "uploads/")
}

// Building handlers

// GetBuildings returns all active buildings
func (h *Handler) GetBuildings(w http.ResponseWriter, r *http.Request) {
	buildings, err := h.repo.GetBuildings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get buildings", err)
		return
	}

	writeJSON(w, http.StatusOK, database.BuildingsResponse{Buildings: buildings})
}

// GetBuildingByID returns a specific building
func (h *Handler) GetBuildingByID(w http.ResponseWriter, r *http.Request) {
	buildingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid building ID", nil)
		return
	}

	building, err := h.repo.GetBuildingByID(r.Context(), buildingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get building", err)
		return
	}
	if building == nil {
		writeError(w, http.StatusNotFound, "Building not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, building)
}

// CreateBuilding creates a new building (admin only)
func (h *Handler) CreateBuilding(w http.ResponseWriter, r *http.Request) {
	var req database.CreateBuildingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "Building name is required", nil)
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		writeError(w, http.StatusBadRequest, "Invalid coordinates", nil)
		return
	}

	building, err := h.repo.CreateBuilding(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create building", err)
		return
	}

	writeJSON(w, http.StatusCreated, building)
}

// Lost & Found Area handlers

// GetLostFoundAreas returns all active lost & found areas
func (h *Handler) GetLostFoundAreas(w http.ResponseWriter, r *http.Request) {
	areas, err := h.repo.GetLostFoundAreas(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get lost & found areas", err)
		return
	}

	writeJSON(w, http.StatusOK, database.LostFoundAreasResponse{Areas: areas})
}

// GetLostFoundAreasByBuilding returns lost & found areas for a specific building
func (h *Handler) GetLostFoundAreasByBuilding(w http.ResponseWriter, r *http.Request) {
	buildingID, err := uuid.Parse(chi.URLParam(r, "buildingId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid building ID", nil)
		return
	}

	areas, err := h.repo.GetLostFoundAreasByBuilding(r.Context(), buildingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get lost & found areas", err)
		return
	}

	writeJSON(w, http.StatusOK, database.LostFoundAreasResponse{Areas: areas})
}

// CreateLostFoundArea creates a new lost & found area (admin only)
func (h *Handler) CreateLostFoundArea(w http.ResponseWriter, r *http.Request) {
	var req database.CreateLostFoundAreaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "Area name is required", nil)
		return
	}

	area, err := h.repo.CreateLostFoundArea(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create lost & found area", err)
		return
	}

	writeJSON(w, http.StatusCreated, area)
}

// User handlers

// GetUserByID returns a specific user
func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	user, err := h.repo.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get user", err)
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "User not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, database.UserResponse{User: *user})
}

// Post handlers

// CreatePost creates a new post
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Failed to parse form", nil)
		return
	}

	// Parse form data
	req := database.CreatePostRequest{
		Type:         r.FormValue("type"),
		Category:     r.FormValue("category"),
		Title:        strings.TrimSpace(r.FormValue("title")),
		Description:  r.FormValue("description"),
		ContactEmail: r.FormValue("contact_email"),
		PosterName:   r.FormValue("poster_name"),
	}

	// Validate required fields against the schema's enums
	if !validTypes[req.Type] {
		writeError(w, http.StatusBadRequest, "type must be 'lost' or 'found'", nil)
		return
	}
	if !validCategories[req.Category] {
		writeError(w, http.StatusBadRequest, "category must be one of: pet, document, item, other", nil)
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required", nil)
		return
	}

	// Parse location
	lat, err := strconv.ParseFloat(r.FormValue("latitude"), 64)
	if err != nil || lat < -90 || lat > 90 {
		writeError(w, http.StatusBadRequest, "Invalid latitude", nil)
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("longitude"), 64)
	if err != nil || lng < -180 || lng > 180 {
		writeError(w, http.StatusBadRequest, "Invalid longitude", nil)
		return
	}
	req.Latitude = lat
	req.Longitude = lng

	// Parse lost & found area ID if provided
	if lostFoundAreaIDStr := r.FormValue("lost_found_area_id"); lostFoundAreaIDStr != "" {
		lostFoundAreaID, err := uuid.Parse(lostFoundAreaIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid lost & found area ID", nil)
			return
		}
		req.LostFoundAreaID = &lostFoundAreaID
	}

	// Parse is_lost_item
	req.IsLostItem = r.FormValue("is_lost_item") == "true"

	// Resolve the acting user (after validation so bad requests fail fast)
	userID, err := h.currentUserID(r)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "No user account available to post as; run migrations to create the default user", err)
		return
	}

	// Process images
	var imageURLs []string
	if files := r.MultipartForm.File["images"]; len(files) > 0 {
		if len(files) > maxImagesPerPost {
			writeError(w, http.StatusBadRequest, "Too many images (maximum 5)", nil)
			return
		}
		for _, file := range files {
			filename, _, err := h.imgProcessor.ProcessUpload(file)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Failed to process image: "+err.Error(), nil)
				return
			}
			imageURLs = append(imageURLs, h.imgProcessor.GetImageURL(filename))
		}
	}

	// Create post. The response is the only place the edit token is ever
	// returned; the client must store it to edit or delete the post later.
	post, err := h.repo.CreatePost(r.Context(), req, userID, imageURLs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create post", err)
		return
	}

	writeJSON(w, http.StatusCreated, post)
}

// SearchPosts searches for posts with various filters
func (h *Handler) SearchPosts(w http.ResponseWriter, r *http.Request) {
	req := database.SearchPostsRequest{}

	// Location parameters. An absent param means "not filtering by it"; a
	// malformed one is a client error, so report it instead of silently
	// falling back to a default the caller never asked for.
	q := r.URL.Query()
	if latStr := q.Get("lat"); latStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil || lat < -90 || lat > 90 {
			writeError(w, http.StatusBadRequest, "lat must be a number between -90 and 90", nil)
			return
		}
		req.Latitude = lat
	}
	if lngStr := q.Get("lng"); lngStr != "" {
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil || lng < -180 || lng > 180 {
			writeError(w, http.StatusBadRequest, "lng must be a number between -180 and 180", nil)
			return
		}
		req.Longitude = lng
	}
	if radiusStr := q.Get("radius"); radiusStr != "" {
		radius, err := strconv.Atoi(radiusStr)
		if err != nil || radius <= 0 {
			writeError(w, http.StatusBadRequest, "radius must be a positive integer (metres)", nil)
			return
		}
		req.Radius = radius
	}

	// Filter parameters. These map to Postgres enum columns, so reject unknown
	// values here rather than letting the driver fail with a 500.
	req.Type = q.Get("type")
	if req.Type != "" && !validTypes[req.Type] {
		writeError(w, http.StatusBadRequest, "type must be 'lost' or 'found'", nil)
		return
	}
	req.Category = q.Get("category")
	if req.Category != "" && !validCategories[req.Category] {
		writeError(w, http.StatusBadRequest, "category must be one of: pet, document, item, other", nil)
		return
	}

	if buildingIDStr := q.Get("building_id"); buildingIDStr != "" {
		buildingID, err := uuid.Parse(buildingIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "building_id must be a valid UUID", nil)
			return
		}
		req.BuildingID = &buildingID
	}
	if areaIDStr := q.Get("lost_found_area_id"); areaIDStr != "" {
		areaID, err := uuid.Parse(areaIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "lost_found_area_id must be a valid UUID", nil)
			return
		}
		req.LostFoundAreaID = &areaID
	}
	if isLostItemStr := q.Get("is_lost_item"); isLostItemStr != "" {
		isLostItem, err := strconv.ParseBool(isLostItemStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "is_lost_item must be true or false", nil)
			return
		}
		req.IsLostItem = &isLostItem
	}

	// Pagination. Values are range-clamped again in the repository; here we
	// only reject input that is outright malformed or nonsensical.
	if limitStr := q.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer", nil)
			return
		}
		req.Limit = limit
	}
	if offsetStr := q.Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "offset must be a non-negative integer", nil)
			return
		}
		req.Offset = offset
	}

	response, err := h.repo.SearchPosts(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to search posts", err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// GetPostByID returns a specific post
func (h *Handler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID", nil)
		return
	}

	post, err := h.repo.GetPostByID(r.Context(), postID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get post", err)
		return
	}
	if post == nil {
		writeError(w, http.StatusNotFound, "Post not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, post)
}

// ClaimPost claims a post for a user
func (h *Handler) ClaimPost(w http.ResponseWriter, r *http.Request) {
	userID, err := h.currentUserID(r)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "No user account available; run migrations to create the default user", err)
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID", nil)
		return
	}

	if err := h.repo.ClaimPost(r.Context(), postID, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Post not found or already claimed", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to claim post", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Post claimed successfully"})
}

// UpdatePost updates a post. Requires the post's edit token via the
// X-Edit-Token header (or edit_token query parameter).
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID", nil)
		return
	}

	editToken := editTokenFromRequest(r)
	if editToken == "" {
		writeError(w, http.StatusUnauthorized, "Edit token required (X-Edit-Token header)", nil)
		return
	}

	var req database.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	if req.Status != "" && !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "status must be one of: active, claimed, resolved", nil)
		return
	}

	if err := h.repo.UpdatePost(r.Context(), postID, editToken, req); err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			writeError(w, http.StatusNotFound, "Post not found", nil)
		case errors.Is(err, database.ErrInvalidEditToken):
			writeError(w, http.StatusForbidden, "Invalid edit token", nil)
		default:
			writeError(w, http.StatusInternalServerError, "Failed to update post", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Post updated successfully"})
}

// DeletePost deletes a post. Requires the post's edit token via the
// X-Edit-Token header (or edit_token query parameter).
func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID", nil)
		return
	}

	editToken := editTokenFromRequest(r)
	if editToken == "" {
		writeError(w, http.StatusUnauthorized, "Edit token required (X-Edit-Token header)", nil)
		return
	}

	imageURLs, err := h.repo.DeletePost(r.Context(), postID, editToken)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			writeError(w, http.StatusNotFound, "Post not found", nil)
		case errors.Is(err, database.ErrInvalidEditToken):
			writeError(w, http.StatusForbidden, "Invalid edit token", nil)
		default:
			writeError(w, http.StatusInternalServerError, "Failed to delete post", err)
		}
		return
	}

	// Best-effort cleanup of the post's image files
	for _, url := range imageURLs {
		if filename := stripUploadsPrefix(url); filename != "" {
			if err := h.imgProcessor.DeleteImage(filename); err != nil {
				log.Printf("failed to delete image %s: %v", filename, err)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Post deleted successfully"})
}

// Interaction handlers

// CreateInteraction records a claim/help/report interaction on a post so the
// poster can review it ("I think this is mine" + contact info).
func (h *Handler) CreateInteraction(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID", nil)
		return
	}

	var req database.CreateInteractionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	req.ContactEmail = strings.TrimSpace(req.ContactEmail)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.Message = strings.TrimSpace(req.Message)

	if !validInteractionTypes[req.InteractionType] {
		writeError(w, http.StatusBadRequest, "interaction_type must be one of: claim, help, report", nil)
		return
	}
	if req.ContactEmail == "" || !strings.Contains(req.ContactEmail, "@") {
		writeError(w, http.StatusBadRequest, "A valid contact_email is required", nil)
		return
	}
	if len(req.Message) > 2000 {
		writeError(w, http.StatusBadRequest, "Message is too long (max 2000 characters)", nil)
		return
	}

	interaction, err := h.repo.CreateInteraction(r.Context(), postID, req)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			writeError(w, http.StatusNotFound, "Post not found", nil)
		case errors.Is(err, database.ErrPostNotActive):
			writeError(w, http.StatusConflict, "This post is no longer active", nil)
		default:
			writeError(w, http.StatusInternalServerError, "Failed to create interaction", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, interaction)
}

// GetPostInteractions lists the interactions on a post. Only the poster (who
// holds the edit token) may view them, since they contain contact details.
func (h *Handler) GetPostInteractions(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID", nil)
		return
	}

	editToken := editTokenFromRequest(r)
	if editToken == "" {
		writeError(w, http.StatusUnauthorized, "Edit token required (X-Edit-Token header)", nil)
		return
	}

	interactions, err := h.repo.GetInteractionsByPost(r.Context(), postID, editToken)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			writeError(w, http.StatusNotFound, "Post not found", nil)
		case errors.Is(err, database.ErrInvalidEditToken):
			writeError(w, http.StatusForbidden, "Invalid edit token", nil)
		default:
			writeError(w, http.StatusInternalServerError, "Failed to get interactions", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, database.InteractionsResponse{Interactions: interactions})
}

// UpdateInteraction lets the poster accept or reject an interaction on their
// post. Accepting a claim marks the post as claimed.
func (h *Handler) UpdateInteraction(w http.ResponseWriter, r *http.Request) {
	interactionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid interaction ID", nil)
		return
	}

	editToken := editTokenFromRequest(r)
	if editToken == "" {
		writeError(w, http.StatusUnauthorized, "Edit token required (X-Edit-Token header)", nil)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	if !validInteractionStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "status must be 'accepted' or 'rejected'", nil)
		return
	}

	if err := h.repo.UpdateInteractionStatus(r.Context(), interactionID, editToken, req.Status); err != nil {
		switch {
		case errors.Is(err, database.ErrNotFound):
			writeError(w, http.StatusNotFound, "Interaction not found", nil)
		case errors.Is(err, database.ErrInvalidEditToken):
			writeError(w, http.StatusForbidden, "Invalid edit token", nil)
		default:
			writeError(w, http.StatusInternalServerError, "Failed to update interaction", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Interaction updated successfully"})
}

// Report handlers

// CreateReport files an abuse report against a post. Open to anyone: the point
// is to hear about bad content from people who are not signed in.
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid post ID", nil)
		return
	}

	var req database.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	if !validReportReasons[req.Reason] {
		writeError(w, http.StatusBadRequest, "reason must be one of: spam, inappropriate, fraudulent, wrong_info, other", nil)
		return
	}
	req.ReporterEmail = strings.TrimSpace(req.ReporterEmail)
	if req.ReporterEmail != "" && !strings.Contains(req.ReporterEmail, "@") {
		writeError(w, http.StatusBadRequest, "reporter_email must be a valid email address", nil)
		return
	}

	report, err := h.repo.CreateReport(r.Context(), postID, req)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Post not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to file report", err)
		return
	}

	writeJSON(w, http.StatusCreated, report)
}

// GetReports lists reports for moderators. Admin only: reports carry the
// reporter's email and accusations about other people's posts.
func (h *Handler) GetReports(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && !validReportStatuses[status] {
		writeError(w, http.StatusBadRequest, "status must be one of: pending, reviewed, resolved", nil)
		return
	}

	reports, err := h.repo.GetReports(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list reports", err)
		return
	}

	writeJSON(w, http.StatusOK, database.ReportsResponse{Reports: reports, Total: len(reports)})
}

// UpdateReport records a moderator's decision on a report.
func (h *Handler) UpdateReport(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid report ID", nil)
		return
	}

	var req database.UpdateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}
	if !validReportStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "status must be one of: pending, reviewed, resolved", nil)
		return
	}

	if err := h.repo.UpdateReportStatus(r.Context(), id, req.Status); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Report not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update report", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

// Alert handlers

// CreateAlert subscribes an email address to posts appearing near a location.
//
// NOTE: this records the subscription. Nothing sends mail yet -- there is no
// dispatcher wired to SMTP -- so alerts accumulate and are never delivered.
func (h *Handler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	var req database.CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "email must be a valid email address", nil)
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		writeError(w, http.StatusBadRequest, "latitude must be between -90 and 90", nil)
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		writeError(w, http.StatusBadRequest, "longitude must be between -180 and 180", nil)
		return
	}
	if req.RadiusMeters <= 0 {
		req.RadiusMeters = 5000
	}
	if req.RadiusMeters > 50000 {
		writeError(w, http.StatusBadRequest, "radius_meters must be 50000 or less", nil)
		return
	}
	for _, c := range req.Categories {
		if !validCategories[c] {
			writeError(w, http.StatusBadRequest, "categories must be among: pet, document, item, other", nil)
			return
		}
	}

	alert, err := h.repo.CreateAlert(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create alert", err)
		return
	}

	writeJSON(w, http.StatusCreated, alert)
}

// GetAlerts lists the alerts for an email address.
func (h *Handler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "email query parameter is required", nil)
		return
	}

	alerts, err := h.repo.GetAlertsByEmail(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list alerts", err)
		return
	}

	writeJSON(w, http.StatusOK, database.AlertsResponse{Alerts: alerts, Total: len(alerts)})
}

// DeleteAlert unsubscribes an alert. The caller must supply the email the
// alert was created with, so an ID on its own does not let someone else
// unsubscribe you.
func (h *Handler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid alert ID", nil)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("email")))
	if email == "" {
		writeError(w, http.StatusBadRequest, "email query parameter is required", nil)
		return
	}

	if err := h.repo.DeactivateAlert(r.Context(), id, email); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Alert not found", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete alert", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
