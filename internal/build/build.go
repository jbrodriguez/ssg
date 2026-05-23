// Package build orchestrates the full site build.
package build

import (
	"fmt"

	"github.com/jbrodriguez/ssg/internal/config"
	"github.com/jbrodriguez/ssg/internal/content"
)

// Options controls runtime behavior of a build.
type Options struct {
	Serve bool
	Watch bool
	Port  int
}

// Run executes the build pipeline.  This is a minimal scaffold; further stages
// (tailwind, render, images, static copy, feeds, serve) are added incrementally.
func Run(cfg *config.Config, opts Options) error {
	posts, err := content.LoadPosts(cfg.PostsDir(), cfg.FilterDrafts)
	if err != nil {
		return err
	}
	fmt.Printf("ssg: loaded %d posts from %s\n", len(posts), cfg.PostsDir())

	tags := content.TagsFromPosts(posts)
	fmt.Printf("ssg: %d unique tags\n", len(tags))

	// TODO: tailwind → render → images → static copy → feeds → serve
	return nil
}
