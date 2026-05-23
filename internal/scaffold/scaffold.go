// Package scaffold creates new posts from a slug + optional title.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbrodriguez/ssg/internal/config"
)

// NewPostOptions controls what frontmatter the scaffolder emits.
type NewPostOptions struct {
	Title string // optional override; defaults to "Notes <slug>"
}

// NewPost creates content/posts/<slug>/index.md with default frontmatter.
// Returns an error if the post already exists.
func NewPost(cfg *config.Config, slug string, opts NewPostOptions) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return fmt.Errorf("slug is required")
	}

	dir := filepath.Join(cfg.PostsDir(), slug)
	mdPath := filepath.Join(dir, "index.md")
	if _, err := os.Stat(mdPath); err == nil {
		return fmt.Errorf("post already exists: %s", mdPath)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	title := opts.Title
	if title == "" {
		title = "Notes " + slug
	}
	date := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	frontmatter := fmt.Sprintf(`---
title: %q
date: %s
cover: ./%s-feature.jpg
caption: " © %s"
status: draft
description: ""
pixelfed: ''
---
`, title, date, slug, cfg.SiteAuthor)

	if err := os.WriteFile(mdPath, []byte(frontmatter), 0o644); err != nil {
		return err
	}
	fmt.Printf("ssg: created %s\n", mdPath)
	return nil
}
