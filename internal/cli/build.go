package cli

import (
	"github.com/jbrodriguez/ssg/internal/build"
	"github.com/jbrodriguez/ssg/internal/config"
)

// BuildCmd is `ssg build`.
type BuildCmd struct {
	SiteRoot string `help:"Override site_root from config." type:"path"`
	Serve    bool   `help:"Run dev server after build."`
	Watch    bool   `help:"Watch for file changes and rebuild."`
	Port     int    `default:"4321" help:"Port for the dev server."`
}

// Run executes the build subcommand.
func (b *BuildCmd) Run(r *Root) error {
	cfg, err := config.Load(r.Config, config.Overrides{SiteRoot: b.SiteRoot})
	if err != nil {
		return err
	}
	return build.Run(cfg, build.Options{Serve: b.Serve, Watch: b.Watch, Port: b.Port})
}
