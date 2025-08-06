package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"lostfound/internal/database"
	"lostfound/internal/image"
)

// Handler provides HTTP handlers for the API
type Handler struct {
	repo        *database.Repository
	imgProcessor *image.Processor
}

// NewHandler creates a new handler instance
func NewHandler(repo *database.Repository, imgProcessor *image.Processor) *Handler {
	return &Handler{
		repo:        repo,
		imgProcessor: imgProcessor,
	}
}

// Building handlers

// GetBuildings returns all active buildings
func (h *Handler) GetBuildings(w http.ResponseWriter, r *http.Request) {
	buildings, err := h.repo.GetBuildings(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get buildings: %v", err), http.StatusInternalServerError)
		return
	}

	response := database.BuildingsResponse{Buildings: buildings}
	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetBuildingByID returns a specific building
func (h *Handler) GetBuildingByID(w http.ResponseWriter, r *http.Request) {
	buildingIDStr := chi.URLParam(r, "id")
	buildingID, err := uuid.Parse(buildingIDStr)
	if err != nil {
		http.Error(w, "Invalid building ID", http.StatusBadRequest)
		return
	}

	building, err := h.repo.GetBuildingByID(r.Context(), buildingID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get building: %v", err), http.StatusInternalServerError)
		return
	}

	if building == nil {
		http.Error(w, "Building not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    building,
	})
}

// CreateBuilding creates a new building (admin only)
func (h *Handler) CreateBuilding(w http.ResponseWriter, r *http.Request) {
	var req database.CreateBuildingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	building, err := h.repo.CreateBuilding(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create building: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    building,
	})
}

// Lost & Found Area handlers

// GetLostFoundAreas returns all active lost & found areas
func (h *Handler) GetLostFoundAreas(w http.ResponseWriter, r *http.Request) {
	areas, err := h.repo.GetLostFoundAreas(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get lost & found areas: %v", err), http.StatusInternalServerError)
		return
	}

	response := database.LostFoundAreasResponse{Areas: areas}
	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetLostFoundAreasByBuilding returns lost & found areas for a specific building
func (h *Handler) GetLostFoundAreasByBuilding(w http.ResponseWriter, r *http.Request) {
	buildingIDStr := chi.URLParam(r, "buildingId")
	buildingID, err := uuid.Parse(buildingIDStr)
	if err != nil {
		http.Error(w, "Invalid building ID", http.StatusBadRequest)
		return
	}

	areas, err := h.repo.GetLostFoundAreasByBuilding(r.Context(), buildingID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get lost & found areas: %v", err), http.StatusInternalServerError)
		return
	}

	response := database.LostFoundAreasResponse{Areas: areas}
	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    response,
	})
}

// CreateLostFoundArea creates a new lost & found area (admin only)
func (h *Handler) CreateLostFoundArea(w http.ResponseWriter, r *http.Request) {
	var req database.CreateLostFoundAreaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	area, err := h.repo.CreateLostFoundArea(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create lost & found area: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    area,
	})
}

// User handlers

// GetOrCreateUser handles SSO user creation/retrieval
func (h *Handler) GetOrCreateUser(w http.ResponseWriter, r *http.Request) {
	var ssoUser database.SSOUser
	if err := json.NewDecoder(r.Body).Decode(&ssoUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetOrCreateUser(r.Context(), ssoUser)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get or create user: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    database.UserResponse{User: *user},
	})
}

// GetUserByID returns a specific user
func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user: %v", err), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    database.UserResponse{User: *user},
	})
}

// Post handlers

// CreatePost creates a new post
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get user ID from context (set by auth middleware) or use default admin user
	var userID uuid.UUID
	if userIDValue := r.Context().Value("user_id"); userIDValue != nil {
		if userIDStr, ok := userIDValue.(string); ok && userIDStr != "" {
			parsedUserID, err := uuid.Parse(userIDStr)
			if err != nil {
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}
			userID = parsedUserID
		} else {
			// Use default admin user for testing
			userID = uuid.MustParse("323746f1-686d-4dd4-a58c-0719ac53db72") // Admin user from sample data
		}
	} else {
		// Use default admin user for testing
		userID = uuid.MustParse("323746f1-686d-4dd4-a58c-0719ac53db72") // Admin user from sample data
	}

	// Parse form data
	req := database.CreatePostRequest{
		Type:        r.FormValue("type"),
		Category:    r.FormValue("category"),
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		ContactEmail: r.FormValue("contact_email"),
		PosterName:   r.FormValue("poster_name"),
	}

	// Parse location
	lat, err := strconv.ParseFloat(r.FormValue("latitude"), 64)
	if err != nil {
		http.Error(w, "Invalid latitude", http.StatusBadRequest)
		return
	}
	lng, err := strconv.ParseFloat(r.FormValue("longitude"), 64)
	if err != nil {
		http.Error(w, "Invalid longitude", http.StatusBadRequest)
		return
	}
	req.Latitude = lat
	req.Longitude = lng

	// Parse lost & found area ID if provided
	if lostFoundAreaIDStr := r.FormValue("lost_found_area_id"); lostFoundAreaIDStr != "" {
		lostFoundAreaID, err := uuid.Parse(lostFoundAreaIDStr)
		if err != nil {
			http.Error(w, "Invalid lost & found area ID", http.StatusBadRequest)
			return
		}
		req.LostFoundAreaID = &lostFoundAreaID
	}

	// Parse is_lost_item
	req.IsLostItem = r.FormValue("is_lost_item") == "true"

		// Process images
	var imageURLs []string
	if files := r.MultipartForm.File["images"]; len(files) > 0 {
		for _, file := range files {
			imageURL, _, err := h.imgProcessor.ProcessUpload(file)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to process image: %v", err), http.StatusInternalServerError)
				return
			}
			imageURLs = append(imageURLs, imageURL)
		}
	}

	// Create post
	post, err := h.repo.CreatePost(r.Context(), req, userID, imageURLs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create post: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    post,
	})
}

// SearchPosts searches for posts with various filters
func (h *Handler) SearchPosts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
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
		if radius, err := strconv.Atoi(radiusStr); err == nil {
			req.Radius = radius
		}
	}

	// Filter parameters
	if req.Type = r.URL.Query().Get("type"); req.Type == "" {
		req.Type = ""
	}
	if req.Category = r.URL.Query().Get("category"); req.Category == "" {
		req.Category = ""
	}

	// Building filter
	if buildingIDStr := r.URL.Query().Get("building_id"); buildingIDStr != "" {
		if buildingID, err := uuid.Parse(buildingIDStr); err == nil {
			req.BuildingID = &buildingID
		}
	}

	// Lost & Found Area filter
	if areaIDStr := r.URL.Query().Get("lost_found_area_id"); areaIDStr != "" {
		if areaID, err := uuid.Parse(areaIDStr); err == nil {
			req.LostFoundAreaID = &areaID
		}
	}

	// Lost/Found item filter
	if isLostItemStr := r.URL.Query().Get("is_lost_item"); isLostItemStr != "" {
		if isLostItem, err := strconv.ParseBool(isLostItemStr); err == nil {
			req.IsLostItem = &isLostItem
		}
	}

	// Pagination
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

	// Search posts
	response, err := h.repo.SearchPosts(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search posts: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetPostByID returns a specific post
func (h *Handler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.repo.GetPostByID(r.Context(), postID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get post: %v", err), http.StatusInternalServerError)
		return
	}

	if post == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    post,
	})
}

// ClaimPost claims a post for a user
func (h *Handler) ClaimPost(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware) or use default admin user
	var userID uuid.UUID
	if userIDValue := r.Context().Value("user_id"); userIDValue != nil {
		if userIDStr, ok := userIDValue.(string); ok && userIDStr != "" {
			parsedUserID, err := uuid.Parse(userIDStr)
			if err != nil {
				http.Error(w, "Invalid user ID", http.StatusBadRequest)
				return
			}
			userID = parsedUserID
		} else {
			// Use default admin user for testing
			userID = uuid.MustParse("323746f1-686d-4dd4-a58c-0719ac53db72") // Admin user from sample data
		}
	} else {
		// Use default admin user for testing
		userID = uuid.MustParse("323746f1-686d-4dd4-a58c-0719ac53db72") // Admin user from sample data
	}

	// Get post ID from URL
	postIDStr := chi.URLParam(r, "id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Claim the post
	err = h.repo.ClaimPost(r.Context(), postID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Post not found or already claimed", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to claim post: %v", err), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Post claimed successfully"},
	})
}

// UpdatePost updates a post
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var req database.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.repo.UpdatePost(r.Context(), postID, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Post not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to update post: %v", err), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Post updated successfully"},
	})
}

// DeletePost deletes a post
func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	err = h.repo.DeletePost(r.Context(), postID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Post not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete post: %v", err), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Post deleted successfully"},
	})
} 