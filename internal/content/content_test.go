package content_test

import (
	"os"
	"testing"

	"github.com/jbrodriguez/ssg/internal/content"
)

// realPostsDir points at the live jbrio.net posts dir.  The test is skipped
// when the directory is not available (e.g. on CI without the site checkout).
const realPostsDir = "/Users/jbrodriguez/.local/share/hosting/cloudflare/jbrio.net/src/data/posts"

func TestLoadPosts_real(t *testing.T) {
	if _, err := os.Stat(realPostsDir); err != nil {
		t.Skipf("real posts dir not available: %v", err)
	}

	posts, err := content.LoadPosts(realPostsDir, true)
	if err != nil {
		t.Fatalf("LoadPosts: %v", err)
	}
	if len(posts) < 200 {
		t.Errorf("expected at least 200 posts, got %d", len(posts))
	}
	t.Logf("loaded %d posts", len(posts))

	// Sorted desc by date.
	for i := 1; i < len(posts); i++ {
		if posts[i-1].Date.Before(posts[i].Date) {
			t.Errorf("posts not sorted desc by date: %s (%s) after %s (%s)",
				posts[i-1].Slug, posts[i-1].Date, posts[i].Slug, posts[i].Date)
			break
		}
	}

	// Every post has Title, Date, and Slug.
	for _, p := range posts {
		if p.Title == "" {
			t.Errorf("post %s missing title", p.Slug)
		}
		if p.Date.IsZero() {
			t.Errorf("post %s has zero date", p.Slug)
		}
		if p.Slug == "" {
			t.Errorf("post has empty slug (path=%s)", p.Path)
		}
		if p.Status == "draft" {
			t.Errorf("draft post %s should have been filtered", p.Slug)
		}
		if want := "/posts/" + p.Slug + "/"; p.URL() != want {
			t.Errorf("URL = %q, want %q", p.URL(), want)
		}
	}

	tags := content.TagsFromPosts(posts)
	if len(tags) == 0 {
		t.Errorf("expected tags, got none")
	}
	t.Logf("found %d unique tags; top 5: %+v", len(tags), tags[:min(5, len(tags))])

	byTag := content.PostsByTag(posts)
	if len(byTag) != len(tags) {
		t.Errorf("PostsByTag has %d entries, TagsFromPosts has %d", len(byTag), len(tags))
	}

	// SimilarPosts: pick the first post, expect some similar (likely shares tags).
	similar := content.SimilarPosts(posts[0], byTag, 5)
	t.Logf("similar to %q (%v): %d hits", posts[0].Title, posts[0].Tags, len(similar))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
