package cli

import "github.com/alecthomas/kong"

// Root is the top-level kong-parsed CLI.
type Root struct {
	Config  string           `short:"c" name:"config" default:"default" help:"Config name under ~/.config/ssg/ (without .toml)."`
	Version kong.VersionFlag `help:"Print version and exit."`

	Build BuildCmd `cmd:"" help:"Build the site."`
	New   NewCmd   `cmd:"" help:"Scaffold a new post."`
}
