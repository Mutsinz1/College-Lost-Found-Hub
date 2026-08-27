package database

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Sentinel errors returned by repository operations so handlers can map
// them to proper HTTP status codes without string matching.
var (
	// ErrNotFound indicates the requested row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrInvalidEditToken indicates the supplied edit token does not match the post's token.
	ErrInvalidEditToken = errors.New("invalid edit token")
	// ErrPostNotActive indicates the post is no longer active (claimed/resolved/expired).
	ErrPostNotActive = errors.New("post is not active")
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

	// Create new user. If a user with this email already exists (e.g. they
	// previously signed in through a different SSO path), relink it instead
	// of failing on the unique email constraint.
	createQuery := `
		INSERT INTO users (sso_id, email, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE SET sso_id = EXCLUDED.sso_id, name = EXCLUDED.name, updated_at = now()
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
	// Clamp pagination
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Build WHERE clause dynamically from the request filters
	where := []string{"p.status = 'active'", "p.expires_at > now()"}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if req.Type != "" {
		where = append(where, "p.type = "+arg(req.Type)+"::post_type")
	}
	if req.Category != "" {
		where = append(where, "p.category = "+arg(req.Category)+"::post_category")
	}
	if req.LostFoundAreaID != nil {
		where = append(where, "p.lost_found_area_id = "+arg(*req.LostFoundAreaID))
	}
	if req.BuildingID != nil {
		where = append(where, "p.lost_found_area_id IN (SELECT id FROM lost_found_areas WHERE building_id = "+arg(*req.BuildingID)+")")
	}
	if req.IsLostItem != nil {
		where = append(where, "p.is_lost_item = "+arg(*req.IsLostItem))
	}

	// Geospatial filter: only applied when a location is provided
	hasGeo := req.Latitude != 0 || req.Longitude != 0
	if hasGeo && req.Radius > 0 {
		where = append(where, fmt.Sprintf(
			"ST_DWithin(p.location, ST_SetSRID(ST_MakePoint(%s, %s), 4326)::geography, %s)",
			arg(req.Longitude), arg(req.Latitude), arg(req.Radius),
		))
	}

	whereSQL := strings.Join(where, " AND ")

	// Count query shares the WHERE clause and args
	countQuery := "SELECT COUNT(*) FROM posts p WHERE " + whereSQL

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count posts: %w", err)
	}

	// Main query gets its own copy of the args so the distance expression
	// and pagination placeholders don't leak into the count query.
	mainArgs := append([]interface{}{}, args...)
	marg := func(v interface{}) string {
		mainArgs = append(mainArgs, v)
		return fmt.Sprintf("$%d", len(mainArgs))
	}

	distanceExpr := "0::float8"
	orderBy := "p.created_at DESC"
	if hasGeo {
		distanceExpr = fmt.Sprintf(
			"ST_Distance(p.location, ST_SetSRID(ST_MakePoint(%s, %s), 4326)::geography)",
			marg(req.Longitude), marg(req.Latitude),
		)
		orderBy = "distance ASC, p.created_at DESC"
	}

	query := `
		SELECT p.id, p.type::text, p.category::text, p.title, p.description,
		       ST_Y(p.location::geometry) as latitude,
		       ST_X(p.location::geometry) as longitude,
		       p.lost_found_area_id, p.posted_by, p.claimed_by, p.claimed_at, p.pickup_scheduled_at, p.picked_up_at,
		       p.is_lost_item, p.status::text, p.contact_email, p.poster_name, p.image_urls, p.expires_at, p.created_at, p.updated_at,
		       ` + distanceExpr + ` as distance
		FROM posts p
		WHERE ` + whereSQL + `
		ORDER BY ` + orderBy + `
		LIMIT ` + marg(req.Limit) + ` OFFSET ` + marg(req.Offset)

	rows, err := r.db.Query(ctx, query, mainArgs...)
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

// GetPostByID retrieves a post by ID, including its lost & found area and
// building (when linked) so the detail view can show pickup information.
func (r *Repository) GetPostByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	query := `
		SELECT p.id, p.type::text, p.category::text, p.title, p.description,
		       ST_Y(p.location::geometry) as latitude,
		       ST_X(p.location::geometry) as longitude,
		       p.lost_found_area_id, p.posted_by, p.claimed_by, p.claimed_at, p.pickup_scheduled_at, p.picked_up_at,
		       p.is_lost_item, p.status::text, p.contact_email, p.poster_name, p.image_urls, p.expires_at, p.created_at, p.updated_at,
		       lfa.id, lfa.building_id, lfa.name, lfa.location_description, lfa.contact_person,
		       lfa.hours_of_operation, lfa.pickup_instructions, lfa.is_active, lfa.created_at, lfa.updated_at,
		       b.id, b.name, b.description,
		       ST_Y(b.location::geometry), ST_X(b.location::geometry),
		       b.is_active, b.created_at, b.updated_at
		FROM posts p
		LEFT JOIN lost_found_areas lfa ON p.lost_found_area_id = lfa.id
		LEFT JOIN buildings b ON lfa.building_id = b.id
		WHERE p.id = $1`

	var post Post
	var (
		lfaID, lfaBuildingID                             *uuid.UUID
		lfaName, lfaLocDesc, lfaContact                  *string
		lfaHours, lfaPickup                              *string
		lfaActive                                        *bool
		lfaCreated, lfaUpdated                           *time.Time
		bID                                              *uuid.UUID
		bName, bDesc                                     *string
		bLat, bLng                                       *float64
		bActive                                          *bool
		bCreated, bUpdated                               *time.Time
	)

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
		&post.ImageURLs,
		&post.ExpiresAt,
		&post.CreatedAt,
		&post.UpdatedAt,
		&lfaID, &lfaBuildingID, &lfaName, &lfaLocDesc, &lfaContact,
		&lfaHours, &lfaPickup, &lfaActive, &lfaCreated, &lfaUpdated,
		&bID, &bName, &bDesc, &bLat, &bLng, &bActive, &bCreated, &bUpdated,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if lfaID != nil {
		area := &LostFoundArea{
			ID:                  *lfaID,
			BuildingID:          derefUUID(lfaBuildingID),
			Name:                derefString(lfaName),
			LocationDescription: derefString(lfaLocDesc),
			ContactPerson:       derefString(lfaContact),
			HoursOfOperation:    derefString(lfaHours),
			PickupInstructions:  derefString(lfaPickup),
			IsActive:            derefBool(lfaActive),
			CreatedAt:           derefTime(lfaCreated),
			UpdatedAt:           derefTime(lfaUpdated),
		}
		if bID != nil {
			area.Building = &Building{
				ID:          *bID,
				Name:        derefString(bName),
				Description: derefString(bDesc),
				Location:    Point{Latitude: derefFloat(bLat), Longitude: derefFloat(bLng)},
				IsActive:    derefBool(bActive),
				CreatedAt:   derefTime(bCreated),
				UpdatedAt:   derefTime(bUpdated),
			}
		}
		post.LostFoundArea = area
	}

	return &post, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func derefUUID(u *uuid.UUID) uuid.UUID {
	if u == nil {
		return uuid.Nil
	}
	return *u
}

// ClaimPost claims a post for a user
func (r *Repository) ClaimPost(ctx context.Context, postID, userID uuid.UUID) error {
	query := `
		UPDATE posts
		SET claimed_by = $1, claimed_at = now(), status = 'claimed'
		WHERE id = $2 AND status = 'active' AND claimed_by IS NULL`

	rows, err := r.db.ExecRows(ctx, query, userID, postID)
	if err != nil {
		return fmt.Errorf("failed to claim post: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// verifyEditToken checks that the supplied edit token matches the post's stored
// token. Returns ErrNotFound if the post does not exist and ErrInvalidEditToken
// if the token does not match. It also returns the post's image URLs so callers
// can clean up files after a delete.
func (r *Repository) verifyEditToken(ctx context.Context, id uuid.UUID, editToken string) ([]string, error) {
	var storedToken string
	var imageURLs []string
	err := r.db.QueryRow(ctx, `SELECT edit_token, image_urls FROM posts WHERE id = $1`, id).Scan(&storedToken, &imageURLs)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to verify edit token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(storedToken), []byte(editToken)) != 1 {
		return nil, ErrInvalidEditToken
	}
	return imageURLs, nil
}

// UpdatePost updates a post after verifying the edit token.
// Empty fields in the request leave the existing values untouched.
func (r *Repository) UpdatePost(ctx context.Context, id uuid.UUID, editToken string, req UpdatePostRequest) error {
	if _, err := r.verifyEditToken(ctx, id, editToken); err != nil {
		return err
	}

	query := `
		UPDATE posts
		SET title = COALESCE(NULLIF($1, ''), title),
		    description = COALESCE(NULLIF($2, ''), description),
		    status = COALESCE(NULLIF($3, '')::post_status, status),
		    updated_at = now()
		WHERE id = $4`

	err := r.db.Exec(ctx, query, req.Title, req.Description, req.Status, id)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

// DeletePost deletes a post after verifying the edit token.
// It returns the post's image URLs so the caller can remove the files from disk.
func (r *Repository) DeletePost(ctx context.Context, id uuid.UUID, editToken string) ([]string, error) {
	imageURLs, err := r.verifyEditToken(ctx, id, editToken)
	if err != nil {
		return nil, err
	}

	if err := r.db.Exec(ctx, `DELETE FROM posts WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("failed to delete post: %w", err)
	}

	return imageURLs, nil
}

// CleanupExpiredPosts deletes expired posts and returns how many were removed
// along with the image URLs of the deleted posts so files can be cleaned up.
func (r *Repository) CleanupExpiredPosts(ctx context.Context) (int, []string, error) {
	rows, err := r.db.Query(ctx, `DELETE FROM posts WHERE expires_at < now() AND status != 'resolved' RETURNING image_urls`)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to cleanup expired posts: %w", err)
	}
	defer rows.Close()

	count := 0
	var images []string
	for rows.Next() {
		var urls []string
		if err := rows.Scan(&urls); err != nil {
			return count, images, fmt.Errorf("failed to scan cleanup row: %w", err)
		}
		count++
		images = append(images, urls...)
	}

	return count, images, nil
}

// Interaction operations

// CreateInteraction records a claim/help/report interaction on a post.
func (r *Repository) CreateInteraction(ctx context.Context, postID uuid.UUID, req CreateInteractionRequest) (*Interaction, error) {
	// Make sure the post exists and is still active
	var status string
	err := r.db.QueryRow(ctx, `SELECT status::text FROM posts WHERE id = $1`, postID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to check post: %w", err)
	}
	if status != "active" {
		return nil, ErrPostNotActive
	}

	query := `
		INSERT INTO interactions (post_id, interaction_type, contact_email, contact_name, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, post_id, interaction_type, COALESCE(contact_email, ''), COALESCE(contact_name, ''), COALESCE(message, ''), created_at, status`

	var interaction Interaction
	err = r.db.QueryRow(ctx, query, postID, req.InteractionType, req.ContactEmail, req.ContactName, req.Message).Scan(
		&interaction.ID,
		&interaction.PostID,
		&interaction.InteractionType,
		&interaction.ContactEmail,
		&interaction.ContactName,
		&interaction.Message,
		&interaction.CreatedAt,
		&interaction.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create interaction: %w", err)
	}

	return &interaction, nil
}

// GetInteractionsByPost lists interactions on a post. The caller must hold the
// post's edit token (i.e. be the poster).
func (r *Repository) GetInteractionsByPost(ctx context.Context, postID uuid.UUID, editToken string) ([]Interaction, error) {
	if _, err := r.verifyEditToken(ctx, postID, editToken); err != nil {
		return nil, err
	}

	query := `
		SELECT id, post_id, interaction_type, COALESCE(contact_email, ''), COALESCE(contact_name, ''), COALESCE(message, ''), created_at, status
		FROM interactions
		WHERE post_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get interactions: %w", err)
	}
	defer rows.Close()

	var interactions []Interaction
	for rows.Next() {
		var interaction Interaction
		if err := rows.Scan(
			&interaction.ID,
			&interaction.PostID,
			&interaction.InteractionType,
			&interaction.ContactEmail,
			&interaction.ContactName,
			&interaction.Message,
			&interaction.CreatedAt,
			&interaction.Status,
		); err != nil {
			return nil, fmt.Errorf("failed to scan interaction: %w", err)
		}
		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

// UpdateInteractionStatus accepts or rejects an interaction. Requires the edit
// token of the post the interaction belongs to. Accepting a claim also marks
// the post as claimed.
func (r *Repository) UpdateInteractionStatus(ctx context.Context, interactionID uuid.UUID, editToken, newStatus string) error {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var postID uuid.UUID
	var interactionType, storedToken string
	err = tx.QueryRow(ctx, `
		SELECT i.post_id, i.interaction_type, p.edit_token
		FROM interactions i
		JOIN posts p ON i.post_id = p.id
		WHERE i.id = $1`, interactionID).Scan(&postID, &interactionType, &storedToken)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get interaction: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(storedToken), []byte(editToken)) != 1 {
		return ErrInvalidEditToken
	}

	if _, err := tx.Exec(ctx, `UPDATE interactions SET status = $1 WHERE id = $2`, newStatus, interactionID); err != nil {
		return fmt.Errorf("failed to update interaction: %w", err)
	}

	// Accepting a claim marks the post as claimed
	if newStatus == "accepted" && interactionType == "claim" {
		if _, err := tx.Exec(ctx, `
			UPDATE posts SET status = 'claimed', claimed_at = now()
			WHERE id = $1 AND status = 'active'`, postID); err != nil {
			return fmt.Errorf("failed to mark post claimed: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDefaultUserID returns the ID of the first active admin user. It is used as
// a fallback author for posts until real authentication is wired up.
func (r *Repository) GetDefaultUserID(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE role = 'admin' AND is_active = true ORDER BY created_at LIMIT 1`).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get default user: %w", err)
	}
	return id, nil
} 