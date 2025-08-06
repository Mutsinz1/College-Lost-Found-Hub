package database

import (
	"time"
	"github.com/google/uuid"
)

// Building represents a college building
type Building struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Location    Point     `json:"location"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LostFoundArea represents a lost & found area within a building
type LostFoundArea struct {
	ID                 uuid.UUID `json:"id"`
	BuildingID         uuid.UUID `json:"building_id"`
	Name               string    `json:"name"`
	LocationDescription string   `json:"location_description"`
	ContactPerson      string    `json:"contact_person"`
	HoursOfOperation   string    `json:"hours_of_operation"`
	PickupInstructions string    `json:"pickup_instructions"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	
	// Joined fields
	Building *Building `json:"building,omitempty"`
}

// User represents a college user (student/staff)
type User struct {
	ID        uuid.UUID `json:"id"`
	SSOID     string    `json:"sso_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Post represents a lost or found item
type Post struct {
	ID               uuid.UUID `json:"id"`
	Type             string    `json:"type"`
	Category         string    `json:"category"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Location         Point     `json:"location"`
	LostFoundAreaID  *uuid.UUID `json:"lost_found_area_id"`
	PostedBy         uuid.UUID `json:"posted_by"`
	ClaimedBy        *uuid.UUID `json:"claimed_by"`
	ClaimedAt        *time.Time `json:"claimed_at"`
	PickupScheduledAt *time.Time `json:"pickup_scheduled_at"`
	PickedUpAt       *time.Time `json:"picked_up_at"`
	IsLostItem       bool      `json:"is_lost_item"`
	Status           string    `json:"status"`
	ContactEmail     *string   `json:"contact_email"`
	PosterName       *string   `json:"poster_name"`
	EditToken        string    `json:"edit_token"`
	ImageURLs        []string  `json:"image_urls"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	
	// Joined fields
	LostFoundArea *LostFoundArea `json:"lost_found_area,omitempty"`
	PostedByUser  *User          `json:"posted_by_user,omitempty"`
	ClaimedByUser *User          `json:"claimed_by_user,omitempty"`
	Distance      float64        `json:"distance,omitempty"` // Distance from search location
}

// Point represents a geographic point
type Point struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Interaction represents user interactions with posts
type Interaction struct {
	ID             uuid.UUID `json:"id"`
	PostID         uuid.UUID `json:"post_id"`
	InteractionType string   `json:"interaction_type"`
	ContactEmail   string    `json:"contact_email"`
	ContactName    string    `json:"contact_name"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
	Status         string    `json:"status"`
}

// Report represents a report about a post
type Report struct {
	ID          uuid.UUID `json:"id"`
	PostID      *uuid.UUID `json:"post_id"`
	ReporterEmail string  `json:"reporter_email"`
	Reason      string    `json:"reason"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`
}

// Alert represents a saved search alert
type Alert struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	Location        Point     `json:"location"`
	RadiusMeters    int       `json:"radius_meters"`
	Categories      []string  `json:"categories"`
	Keywords        []string  `json:"keywords"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	LastTriggeredAt *time.Time `json:"last_triggered_at"`
}

// Request/Response types

// CreatePostRequest represents a request to create a new post
type CreatePostRequest struct {
	Type             string    `json:"type"`
	Category         string    `json:"category"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	LostFoundAreaID  *uuid.UUID `json:"lost_found_area_id"`
	IsLostItem       bool      `json:"is_lost_item"`
	ContactEmail     string    `json:"contact_email"`
	PosterName       string    `json:"poster_name"`
}

// UpdatePostRequest represents a request to update a post
type UpdatePostRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// SearchPostsRequest represents a request to search posts
type SearchPostsRequest struct {
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	Radius          int       `json:"radius"`
	Type            string    `json:"type"`
	Category        string    `json:"category"`
	BuildingID      *uuid.UUID `json:"building_id"`
	LostFoundAreaID *uuid.UUID `json:"lost_found_area_id"`
	IsLostItem      *bool     `json:"is_lost_item"`
	Limit           int       `json:"limit"`
	Offset          int       `json:"offset"`
}

// ClaimPostRequest represents a request to claim a post
type ClaimPostRequest struct {
	PostID uuid.UUID `json:"post_id"`
}

// CreateBuildingRequest represents a request to create a building
type CreateBuildingRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// CreateLostFoundAreaRequest represents a request to create a lost & found area
type CreateLostFoundAreaRequest struct {
	BuildingID          uuid.UUID `json:"building_id"`
	Name                string    `json:"name"`
	LocationDescription string    `json:"location_description"`
	ContactPerson       string    `json:"contact_person"`
	HoursOfOperation    string    `json:"hours_of_operation"`
	PickupInstructions  string    `json:"pickup_instructions"`
}

// SSOUser represents user data from SSO
type SSOUser struct {
	SSOID string `json:"sso_id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// API Response types

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SearchPostsResponse represents the response for post search
type SearchPostsResponse struct {
	Posts  []Post `json:"posts"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// BuildingsResponse represents the response for buildings
type BuildingsResponse struct {
	Buildings []Building `json:"buildings"`
}

// LostFoundAreasResponse represents the response for lost & found areas
type LostFoundAreasResponse struct {
	Areas []LostFoundArea `json:"areas"`
}

// UserResponse represents the response for user data
type UserResponse struct {
	User User `json:"user"`
} 