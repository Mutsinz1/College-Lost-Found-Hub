package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These tests need a real PostgreSQL database with PostGIS and the project
// migrations applied. Set TEST_DATABASE_URL to run them; they are skipped
// otherwise. CI provides a postgis service container for this.

func testRepo(t *testing.T) *Repository {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration tests")
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(pool.Close)

	return NewRepository(&DB{pool: pool})
}

func testUserID(t *testing.T, repo *Repository) uuid.UUID {
	t.Helper()
	id, err := repo.GetDefaultUserID(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultUserID failed (did migrations run?): %v", err)
	}
	return id
}

func createTestPost(t *testing.T, repo *Repository) *Post {
	t.Helper()
	post, err := repo.CreatePost(context.Background(), CreatePostRequest{
		Type:        "found",
		Category:    "item",
		Title:       "integration-test item",
		Description: "created by repository_integration_test",
		Latitude:    40.7306,
		Longitude:   -73.9352,
		IsLostItem:  false,
	}, testUserID(t, repo), nil)
	if err != nil {
		t.Fatalf("CreatePost failed: %v", err)
	}
	t.Cleanup(func() {
		// Best-effort cleanup; the post may already be deleted by the test
		_, _ = repo.DeletePost(context.Background(), post.ID, post.EditToken)
	})
	return post
}

func TestPostLifecycle(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	post := createTestPost(t, repo)
	if post.EditToken == "" {
		t.Fatal("CreatePost should return an edit token")
	}

	// Read it back: edit token must not be exposed, area join must not break
	got, err := repo.GetPostByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetPostByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetPostByID returned nil for an existing post")
	}
	if got.EditToken != "" {
		t.Error("GetPostByID must not return the edit token")
	}

	// Update with a wrong token must fail
	if err := repo.UpdatePost(ctx, post.ID, "wrong-token", UpdatePostRequest{Title: "hacked"}); err != ErrInvalidEditToken {
		t.Errorf("UpdatePost with wrong token: err = %v, want ErrInvalidEditToken", err)
	}

	// Update with the right token succeeds and keeps empty fields unchanged
	if err := repo.UpdatePost(ctx, post.ID, post.EditToken, UpdatePostRequest{Title: "updated title"}); err != nil {
		t.Fatalf("UpdatePost failed: %v", err)
	}
	got, err = repo.GetPostByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetPostByID after update failed: %v", err)
	}
	if got.Title != "updated title" {
		t.Errorf("Title = %q, want %q", got.Title, "updated title")
	}
	if got.Description != post.Description {
		t.Errorf("empty update field overwrote description: %q", got.Description)
	}

	// Search with a radius that covers the post's location
	res, err := repo.SearchPosts(ctx, SearchPostsRequest{
		Latitude:  40.7306,
		Longitude: -73.9352,
		Radius:    1000,
		Type:      "found",
		Category:  "item",
	})
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}
	found := false
	for _, p := range res.Posts {
		if p.ID == post.ID {
			found = true
			if p.EditToken != "" {
				t.Error("SearchPosts must not return edit tokens")
			}
		}
	}
	if !found {
		t.Error("SearchPosts did not return the created post within radius")
	}

	// A far-away radius search must not include it
	res, err = repo.SearchPosts(ctx, SearchPostsRequest{
		Latitude:  48.8566, // Paris
		Longitude: 2.3522,
		Radius:    1000,
	})
	if err != nil {
		t.Fatalf("far SearchPosts failed: %v", err)
	}
	for _, p := range res.Posts {
		if p.ID == post.ID {
			t.Error("SearchPosts returned a post far outside the radius")
		}
	}

	// Delete with wrong token fails, right token succeeds
	if _, err := repo.DeletePost(ctx, post.ID, "wrong-token"); err != ErrInvalidEditToken {
		t.Errorf("DeletePost with wrong token: err = %v, want ErrInvalidEditToken", err)
	}
	if _, err := repo.DeletePost(ctx, post.ID, post.EditToken); err != nil {
		t.Fatalf("DeletePost failed: %v", err)
	}
	got, err = repo.GetPostByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetPostByID after delete failed: %v", err)
	}
	if got != nil {
		t.Error("post still exists after delete")
	}
}

func TestInteractionFlow(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	post := createTestPost(t, repo)

	interaction, err := repo.CreateInteraction(ctx, post.ID, CreateInteractionRequest{
		InteractionType: "claim",
		ContactEmail:    "claimer@college.edu",
		ContactName:     "Claimer",
		Message:         "That's my backpack",
	})
	if err != nil {
		t.Fatalf("CreateInteraction failed: %v", err)
	}
	if interaction.Status != "pending" {
		t.Errorf("new interaction status = %q, want pending", interaction.Status)
	}

	// Listing requires the edit token
	if _, err := repo.GetInteractionsByPost(ctx, post.ID, "wrong-token"); err != ErrInvalidEditToken {
		t.Errorf("GetInteractionsByPost with wrong token: err = %v, want ErrInvalidEditToken", err)
	}
	interactions, err := repo.GetInteractionsByPost(ctx, post.ID, post.EditToken)
	if err != nil {
		t.Fatalf("GetInteractionsByPost failed: %v", err)
	}
	if len(interactions) != 1 {
		t.Fatalf("interactions = %d, want 1", len(interactions))
	}

	// Accepting the claim marks the post as claimed
	if err := repo.UpdateInteractionStatus(ctx, interaction.ID, post.EditToken, "accepted"); err != nil {
		t.Fatalf("UpdateInteractionStatus failed: %v", err)
	}
	got, err := repo.GetPostByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetPostByID failed: %v", err)
	}
	if got.Status != "claimed" {
		t.Errorf("post status = %q, want claimed", got.Status)
	}

	// New claims on a non-active post are rejected
	if _, err := repo.CreateInteraction(ctx, post.ID, CreateInteractionRequest{
		InteractionType: "claim",
		ContactEmail:    "late@college.edu",
	}); err != ErrPostNotActive {
		t.Errorf("CreateInteraction on claimed post: err = %v, want ErrPostNotActive", err)
	}
}

func TestClaimPostRowsAffected(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Claiming a nonexistent post reports ErrNotFound
	if err := repo.ClaimPost(ctx, uuid.New(), testUserID(t, repo)); err != ErrNotFound {
		t.Errorf("ClaimPost on missing post: err = %v, want ErrNotFound", err)
	}

	post := createTestPost(t, repo)
	userID := testUserID(t, repo)

	if err := repo.ClaimPost(ctx, post.ID, userID); err != nil {
		t.Fatalf("ClaimPost failed: %v", err)
	}
	// Second claim must fail: the post is no longer active
	if err := repo.ClaimPost(ctx, post.ID, userID); err != ErrNotFound {
		t.Errorf("double claim: err = %v, want ErrNotFound", err)
	}
}

func TestGetOrCreateUserRefusesPrivilegedRelink(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// An operator-provisioned admin row, of the shape the dev seed creates:
	// a role but no usable SSO identity.
	email := fmt.Sprintf("admin-%s@college.edu", uuid.NewString()[:8])
	if err := repo.db.Exec(ctx,
		`INSERT INTO users (sso_id, email, name, role) VALUES ($1, $2, $3, 'admin')`,
		"provisioned_"+uuid.NewString(), email, "Provisioned Admin"); err != nil {
		t.Fatalf("failed to seed admin: %v", err)
	}

	// Someone signs in with a fresh SSO identity that happens to carry the same
	// address. They must not inherit the admin account.
	_, err := repo.GetOrCreateUser(ctx, SSOUser{
		SSOID: "google:" + uuid.NewString(),
		Email: email,
		Name:  "Attacker",
	})
	if !errors.Is(err, ErrPrivilegedRelink) {
		t.Fatalf("err = %v, want ErrPrivilegedRelink", err)
	}

	// The admin row must be untouched.
	var ssoID, role string
	if err := repo.db.QueryRow(ctx, `SELECT sso_id, role FROM users WHERE email = $1`, email).Scan(&ssoID, &role); err != nil {
		t.Fatalf("failed to re-read admin: %v", err)
	}
	if role != "admin" || !strings.HasPrefix(ssoID, "provisioned_") {
		t.Errorf("admin row was modified: sso_id=%q role=%q", ssoID, role)
	}
}

func TestGetOrCreateUserRelinksOrdinaryAccount(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Relinking a normal account across SSO paths must still work; that is the
	// behaviour the privileged check is carving an exception out of.
	email := fmt.Sprintf("student-%s@college.edu", uuid.NewString()[:8])
	first, err := repo.GetOrCreateUser(ctx, SSOUser{SSOID: "dev:" + email, Email: email, Name: "Student"})
	if err != nil {
		t.Fatalf("first sign-in failed: %v", err)
	}

	newSSO := "google:" + uuid.NewString()
	second, err := repo.GetOrCreateUser(ctx, SSOUser{SSOID: newSSO, Email: email, Name: "Student"})
	if err != nil {
		t.Fatalf("relink failed: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("relink created a new row: %s vs %s", second.ID, first.ID)
	}
	if second.SSOID != newSSO {
		t.Errorf("sso_id = %q, want %q", second.SSOID, newSSO)
	}
	if second.Role != "user" {
		t.Errorf("role = %q, want user", second.Role)
	}
}

func TestGetDefaultUserIDIsUnprivileged(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Anonymous posts are attributed to this account. It must never be an
	// admin: an earlier version resolved it with `WHERE role = 'admin'`, which
	// attributed anonymous content to whatever privileged row existed.
	id, err := repo.GetDefaultUserID(ctx)
	if err != nil {
		t.Fatalf("GetDefaultUserID failed (did migration 002 run?): %v", err)
	}

	var ssoID, role string
	if err := repo.db.QueryRow(ctx, `SELECT sso_id, role FROM users WHERE id = $1`, id).Scan(&ssoID, &role); err != nil {
		t.Fatalf("failed to read the default user: %v", err)
	}
	if ssoID != SystemAnonymousSSOID {
		t.Errorf("sso_id = %q, want %q", ssoID, SystemAnonymousSSOID)
	}
	if role != "user" {
		t.Errorf("role = %q, want user: anonymous posts must not be attributed to a privileged account", role)
	}
}

func TestGetDefaultUserIDIgnoresAdmins(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	// Adding an admin must not change who owns anonymous posts.
	before, err := repo.GetDefaultUserID(ctx)
	if err != nil {
		t.Fatalf("GetDefaultUserID failed: %v", err)
	}

	email := fmt.Sprintf("extra-admin-%s@college.edu", uuid.NewString()[:8])
	if err := repo.db.Exec(ctx,
		`INSERT INTO users (sso_id, email, name, role) VALUES ($1, $2, $3, 'admin')`,
		"provisioned_"+uuid.NewString(), email, "Extra Admin"); err != nil {
		t.Fatalf("failed to insert admin: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.db.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	after, err := repo.GetDefaultUserID(ctx)
	if err != nil {
		t.Fatalf("GetDefaultUserID failed after adding an admin: %v", err)
	}
	if after != before {
		t.Errorf("default user changed from %s to %s when an admin was added", before, after)
	}
}
