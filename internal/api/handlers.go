package api

import (
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

var validCategories = map[string]bool{"pet": true, "document": true, "item": true, "other": true}
var validStatuses = map[string]bool{"active": true, "claimed": true, "resolved": true}
var validInteractionTypes = map[string]bool{"claim": true, "help": true, "report": true}
var validInteractionStatuses = map[string]bool{"accepted": true, "rejected": true}

// Handler provides HTTP handlers for the API
type Handler struct {
	repo         *database.Repository
	imgProcessor *image.Processor

	defaultUserOnce sync.Once
	defaultUserID   uuid.UUID
	defaultUserErr  error
}

// NewHandler creates a new handler instance
func NewHandler(repo *database.Repository, imgProcessor *image.Processor) *Handler {
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

	userID, err := h.currentUserID(r)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "No user account available to post as; run migrations to create the default user", err)
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
	if req.Type != "lost" && req.Type != "found" {
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

	// Location parameters
	if latStr := r.URL.Query().Get("lat"); latStr != "" {
		if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
			req.Latitude = lat
		}
	}
	if lngStr := r.URL.Query().Get("lng"); lngStr != "" {
		if lng, err := strconv.ParseFloat(lngStr, 64); err == nil {
			req.Longitude = lng
		}
	}
	if radiusStr := r.URL.Query().Get("radius"); radiusStr != "" {
		if radius, err := strconv.Atoi(radiusStr); err == nil && radius > 0 {
			req.Radius = radius
		}
	}

	// Filter parameters
	req.Type = r.URL.Query().Get("type")
	req.Category = r.URL.Query().Get("category")

	if buildingIDStr := r.URL.Query().Get("building_id"); buildingIDStr != "" {
		if buildingID, err := uuid.Parse(buildingIDStr); err == nil {
			req.BuildingID = &buildingID
		}
	}
	if areaIDStr := r.URL.Query().Get("lost_found_area_id"); areaIDStr != "" {
		if areaID, err := uuid.Parse(areaIDStr); err == nil {
			req.LostFoundAreaID = &areaID
		}
	}
	if isLostItemStr := r.URL.Query().Get("is_lost_item"); isLostItemStr != "" {
		if isLostItem, err := strconv.ParseBool(isLostItemStr); err == nil {
			req.IsLostItem = &isLostItem
		}
	}

	// Pagination (clamped again in the repository)
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = offset
		}
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
