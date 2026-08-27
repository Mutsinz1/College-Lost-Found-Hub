package database

import (
	"context"
	"os"
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
