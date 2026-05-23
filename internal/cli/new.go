package cli

import (
	"github.com/jbrodriguez/ssg/internal/config"
	"github.com/jbrodriguez/ssg/internal/scaffold"
)

// NewCmd is `ssg new <title>`.
type NewCmd struct {
	Title string `arg:"" help:"Title of the new post."`
}

// Run executes the new-post scaffolder.
func (n *NewCmd) Run(r *Root) error {
	cfg, err := config.Load(r.Config, config.Overrides{})
	if err != nil {
		return err
	}
	return scaffold.NewPost(cfg, n.Title)
}
