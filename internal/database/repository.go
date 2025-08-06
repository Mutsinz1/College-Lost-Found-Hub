package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository provides database operations
type Repository struct {
	db *DB
}

// NewRepository creates a new repository instance
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// Building operations

// GetBuildings retrieves all active buildings
func (r *Repository) GetBuildings(ctx context.Context) ([]Building, error) {
	query := `
		SELECT id, name, description, 
		       ST_Y(location::geometry) as latitude, 
		       ST_X(location::geometry) as longitude,
		       is_active, created_at, updated_at
		FROM buildings 
		WHERE is_active = true 
		ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get buildings: %w", err)
	}
	defer rows.Close()

	var buildings []Building
	for rows.Next() {
		var building Building
		err := rows.Scan(
			&building.ID,
			&building.Name,
			&building.Description,
			&building.Location.Latitude,
			&building.Location.Longitude,
			&building.IsActive,
			&building.CreatedAt,
			&building.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan building: %w", err)
		}
		buildings = append(buildings, building)
	}

	return buildings, nil
}

// GetBuildingByID retrieves a building by ID
func (r *Repository) GetBuildingByID(ctx context.Context, id uuid.UUID) (*Building, error) {
	query := `
		SELECT id, name, description, 
		       ST_Y(location::geometry) as latitude, 
		       ST_X(location::geometry) as longitude,
		       is_active, created_at, updated_at
		FROM buildings 
		WHERE id = $1 AND is_active = true`

	var building Building
	err := r.db.QueryRow(ctx, query, id).Scan(
		&building.ID,
		&building.Name,
		&building.Description,
		&building.Location.Latitude,
		&building.Location.Longitude,
		&building.IsActive,
		&building.CreatedAt,
		&building.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get building: %w", err)
	}

	return &building, nil
}

// CreateBuilding creates a new building
func (r *Repository) CreateBuilding(ctx context.Context, req CreateBuildingRequest) (*Building, error) {
	query := `
		INSERT INTO buildings (name, description, location)
		VALUES ($1, $2, ST_GeomFromText($3, 4326))
		RETURNING id, name, description, 
		          ST_Y(location::geometry) as latitude, 
		          ST_X(location::geometry) as longitude,
		          is_active, created_at, updated_at`

	pointText := fmt.Sprintf("POINT(%f %f)", req.Longitude, req.Latitude)
	
	var building Building
	err := r.db.QueryRow(ctx, query, req.Name, req.Description, pointText).Scan(
		&building.ID,
		&building.Name,
		&building.Description,
		&building.Location.Latitude,
		&building.Location.Longitude,
		&building.IsActive,
		&building.CreatedAt,
		&building.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create building: %w", err)
	}

	return &building, nil
}

// Lost & Found Area operations

// GetLostFoundAreas retrieves all active lost & found areas
func (r *Repository) GetLostFoundAreas(ctx context.Context) ([]LostFoundArea, error) {
	query := `
		SELECT lfa.id, lfa.building_id, lfa.name, lfa.location_description,
		       lfa.contact_person, lfa.hours_of_operation, lfa.pickup_instructions,
		       lfa.is_active, lfa.created_at, lfa.updated_at,
		       b.id, b.name, b.description, 
		       ST_Y(b.location::geometry) as latitude, 
		       ST_X(b.location::geometry) as longitude,
		       b.is_active, b.created_at, b.updated_at
		FROM lost_found_areas lfa
		JOIN buildings b ON lfa.building_id = b.id
		WHERE lfa.is_active = true AND b.is_active = true
		ORDER BY b.name, lfa.name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get lost & found areas: %w", err)
	}
	defer rows.Close()

	var areas []LostFoundArea
	for rows.Next() {
		var area LostFoundArea
		var building Building
		err := rows.Scan(
			&area.ID,
			&area.BuildingID,
			&area.Name,
			&area.LocationDescription,
			&area.ContactPerson,
			&area.HoursOfOperation,
			&area.PickupInstructions,
			&area.IsActive,
			&area.CreatedAt,
			&area.UpdatedAt,
			&building.ID,
			&building.Name,
			&building.Description,
			&building.Location.Latitude,
			&building.Location.Longitude,
			&building.IsActive,
			&building.CreatedAt,
			&building.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lost & found area: %w", err)
		}
		area.Building = &building
		areas = append(areas, area)
	}

	return areas, nil
}

// GetLostFoundAreasByBuilding retrieves lost & found areas for a specific building
func (r *Repository) GetLostFoundAreasByBuilding(ctx context.Context, buildingID uuid.UUID) ([]LostFoundArea, error) {
	query := `
		SELECT lfa.id, lfa.building_id, lfa.name, lfa.location_description,
		       lfa.contact_person, lfa.hours_of_operation, lfa.pickup_instructions,
		       lfa.is_active, lfa.created_at, lfa.updated_at,
		       b.id, b.name, b.description, 
		       ST_Y(b.location::geometry) as latitude, 
		       ST_X(b.location::geometry) as longitude,
		       b.is_active, b.created_at, b.updated_at
		FROM lost_found_areas lfa
		JOIN buildings b ON lfa.building_id = b.id
		WHERE lfa.building_id = $1 AND lfa.is_active = true AND b.is_active = true
		ORDER BY lfa.name`

	rows, err := r.db.Query(ctx, query, buildingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lost & found areas: %w", err)
	}
	defer rows.Close()

	var areas []LostFoundArea
	for rows.Next() {
		var area LostFoundArea
		var building Building
		err := rows.Scan(
			&area.ID,
			&area.BuildingID,
			&area.Name,
			&area.LocationDescription,
			&area.ContactPerson,
			&area.HoursOfOperation,
			&area.PickupInstructions,
			&area.IsActive,
			&area.CreatedAt,
			&area.UpdatedAt,
			&building.ID,
			&building.Name,
			&building.Description,
			&building.Location.Latitude,
			&building.Location.Longitude,
			&building.IsActive,
			&building.CreatedAt,
			&building.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lost & found area: %w", err)
		}
		area.Building = &building
		areas = append(areas, area)
	}

	return areas, nil
}

// CreateLostFoundArea creates a new lost & found area
func (r *Repository) CreateLostFoundArea(ctx context.Context, req CreateLostFoundAreaRequest) (*LostFoundArea, error) {
	query := `
		INSERT INTO lost_found_areas (building_id, name, location_description, contact_person, hours_of_operation, pickup_instructions)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, building_id, name, location_description, contact_person, hours_of_operation, pickup_instructions, is_active, created_at, updated_at`

	var area LostFoundArea
	err := r.db.QueryRow(ctx, query, 
		req.BuildingID, req.Name, req.LocationDescription, 
		req.ContactPerson, req.HoursOfOperation, req.PickupInstructions,
	).Scan(
		&area.ID,
		&area.BuildingID,
		&area.Name,
		&area.LocationDescription,
		&area.ContactPerson,
		&area.HoursOfOperation,
		&area.PickupInstructions,
		&area.IsActive,
		&area.CreatedAt,
		&area.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create lost & found area: %w", err)
	}

	return &area, nil
}

// User operations

// GetOrCreateUser retrieves a user by SSO ID or creates a new one
func (r *Repository) GetOrCreateUser(ctx context.Context, ssoUser SSOUser) (*User, error) {
	// Try to get existing user
	query := `
		SELECT id, sso_id, email, name, role, is_active, created_at, updated_at
		FROM users 
		WHERE sso_id = $1`

	var user User
	err := r.db.QueryRow(ctx, query, ssoUser.SSOID).Scan(
		&user.ID,
		&user.SSOID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == nil {
		return &user, nil
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Create new user
	createQuery := `
		INSERT INTO users (sso_id, email, name)
		VALUES ($1, $2, $3)
		RETURNING id, sso_id, email, name, role, is_active, created_at, updated_at`

	err = r.db.QueryRow(ctx, createQuery, ssoUser.SSOID, ssoUser.Email, ssoUser.Name).Scan(
		&user.ID,
		&user.SSOID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, sso_id, email, name, role, is_active, created_at, updated_at
		FROM users 
		WHERE id = $1 AND is_active = true`

	var user User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.SSOID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Post operations

// CreatePost creates a new post
func (r *Repository) CreatePost(ctx context.Context, req CreatePostRequest, userID uuid.UUID, imageURLs []string) (*Post, error) {
	query := `
		INSERT INTO posts (type, category, title, description, location, lost_found_area_id, posted_by, is_lost_item, contact_email, poster_name, edit_token, image_urls)
		VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326), $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, type, category, title, description, 
		          ST_Y(location::geometry) as latitude, 
		          ST_X(location::geometry) as longitude,
		          lost_found_area_id, posted_by, claimed_by, claimed_at, pickup_scheduled_at, picked_up_at,
		          is_lost_item, status, contact_email, poster_name, edit_token, image_urls, expires_at, created_at, updated_at`

	pointText := fmt.Sprintf("POINT(%f %f)", req.Longitude, req.Latitude)
	editToken := uuid.New().String()

	var post Post
	err := r.db.QueryRow(ctx, query, 
		req.Type, req.Category, req.Title, req.Description, pointText,
		req.LostFoundAreaID, userID, req.IsLostItem, req.ContactEmail, req.PosterName,
		editToken, imageURLs,
	).Scan(
		&post.ID,
		&post.Type,
		&post.Category,
		&post.Title,
		&post.Description,
		&post.Location.Latitude,
		&post.Location.Longitude,
		&post.LostFoundAreaID,
		&post.PostedBy,
		&post.ClaimedBy,
		&post.ClaimedAt,
		&post.PickupScheduledAt,
		&post.PickedUpAt,
		&post.IsLostItem,
		&post.Status,
		&post.ContactEmail,
		&post.PosterName,
		&post.EditToken,
		&post.ImageURLs,
		&post.ExpiresAt,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return &post, nil
}

// SearchPosts searches for posts with various filters
func (r *Repository) SearchPosts(ctx context.Context, req SearchPostsRequest) (*SearchPostsResponse, error) {
	// Set default limit
	if req.Limit <= 0 {
		req.Limit = 50
	}

	// Simple count query
	countQuery := `
		SELECT COUNT(*)
		FROM posts p
		WHERE p.status = 'active' AND p.expires_at > now()`

	var total int
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count posts: %w", err)
	}

	// Simple main query without parameters
	query := `
		SELECT p.id, p.type::text, p.category::text, p.title, p.description,
		       ST_Y(p.location::geometry) as latitude, 
		       ST_X(p.location::geometry) as longitude,
		       p.lost_found_area_id, p.posted_by, p.claimed_by, p.claimed_at, p.pickup_scheduled_at, p.picked_up_at,
		       p.is_lost_item, p.status::text, p.contact_email, p.poster_name, p.edit_token, p.image_urls, p.expires_at, p.created_at, p.updated_at,
		       0 as distance
		FROM posts p
		WHERE p.status = 'active' AND p.expires_at > now()
		ORDER BY p.created_at DESC
		LIMIT 50 OFFSET 0`

	args := []interface{}{}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.Type,
			&post.Category,
			&post.Title,
			&post.Description,
			&post.Location.Latitude,
			&post.Location.Longitude,
			&post.LostFoundAreaID,
			&post.PostedBy,
			&post.ClaimedBy,
			&post.ClaimedAt,
			&post.PickupScheduledAt,
			&post.PickedUpAt,
			&post.IsLostItem,
			&post.Status,
			&post.ContactEmail,
			&post.PosterName,
			&post.EditToken,
			&post.ImageURLs,
			&post.ExpiresAt,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.Distance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, post)
	}

	return &SearchPostsResponse{
		Posts:  posts,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

// GetPostByID retrieves a post by ID
func (r *Repository) GetPostByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	query := `
		SELECT p.id, p.type, p.category, p.title, p.description,
		       ST_Y(p.location::geometry) as latitude, 
		       ST_X(p.location::geometry) as longitude,
		       p.lost_found_area_id, p.posted_by, p.claimed_by, p.claimed_at, p.pickup_scheduled_at, p.picked_up_at,
		       p.is_lost_item, p.status, p.contact_email, p.poster_name, p.edit_token, p.image_urls, p.expires_at, p.created_at, p.updated_at
		FROM posts p
		WHERE p.id = $1`

	var post Post
	err := r.db.QueryRow(ctx, query, id).Scan(
		&post.ID,
		&post.Type,
		&post.Category,
		&post.Title,
		&post.Description,
		&post.Location.Latitude,
		&post.Location.Longitude,
		&post.LostFoundAreaID,
		&post.PostedBy,
		&post.ClaimedBy,
		&post.ClaimedAt,
		&post.PickupScheduledAt,
		&post.PickedUpAt,
		&post.IsLostItem,
		&post.Status,
		&post.ContactEmail,
		&post.PosterName,
		&post.EditToken,
		&post.ImageURLs,
		&post.ExpiresAt,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return &post, nil
}

// ClaimPost claims a post for a user
func (r *Repository) ClaimPost(ctx context.Context, postID, userID uuid.UUID) error {
	query := `
		UPDATE posts 
		SET claimed_by = $1, claimed_at = now(), status = 'claimed'
		WHERE id = $2 AND status = 'active' AND claimed_by IS NULL`

	err := r.db.Exec(ctx, query, userID, postID)
	if err != nil {
		return fmt.Errorf("failed to claim post: %w", err)
	}

	return nil
}

// UpdatePost updates a post
func (r *Repository) UpdatePost(ctx context.Context, id uuid.UUID, req UpdatePostRequest) error {
	query := `
		UPDATE posts 
		SET title = $1, description = $2, status = $3, updated_at = now()
		WHERE id = $4`

	err := r.db.Exec(ctx, query, req.Title, req.Description, req.Status, id)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

// DeletePost deletes a post
func (r *Repository) DeletePost(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM posts WHERE id = $1`

	err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	return nil
}

// Cleanup expired posts
func (r *Repository) CleanupExpiredPosts(ctx context.Context) (int, error) {
	query := `DELETE FROM posts WHERE expires_at < now() AND status != 'resolved'`
	err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired posts: %w", err)
	}

	return 0, nil
} 