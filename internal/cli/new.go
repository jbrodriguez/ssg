package cli

import (
	"github.com/jbrodriguez/ssg/internal/config"
	"github.com/jbrodriguez/ssg/internal/scaffold"
)

// NewCmd is `ssg new <slug>`.
type NewCmd struct {
	Slug  string `arg:"" help:"Slug (directory name) for the new post."`
	Title string `short:"t" help:"Post title. Defaults to \"Notes <slug>\"."`
}

// Run executes the new-post scaffolder.
func (n *NewCmd) Run(_ *Root) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	return scaffold.NewPost(cfg, n.Slug, scaffold.NewPostOptions{Title: n.Title})
}
