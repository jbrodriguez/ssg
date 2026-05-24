package cli

import "github.com/alecthomas/kong"

// Root is the top-level kong-parsed CLI.  ssg looks for ssg.toml in the
// current working directory; there is no --config flag.
type Root struct {
	Version kong.VersionFlag `help:"Print version and exit."`

	Build BuildCmd `cmd:"" help:"Build the site (run from the directory containing ssg.toml)."`
	New   NewCmd   `cmd:"" help:"Scaffold a new post."`
	Init  InitCmd  `cmd:"" help:"Write a default ssg.toml in the current directory."`
}
