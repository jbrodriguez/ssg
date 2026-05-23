package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/jbrodriguez/ssg/internal/cli"
)

var version = "dev"

func main() {
	var c cli.Root
	ctx := kong.Parse(&c,
		kong.Name("ssg"),
		kong.Description("A small static site generator."),
		kong.UsageOnError(),
		kong.Vars{"version": version},
	)
	if err := ctx.Run(&c); err != nil {
		fmt.Fprintln(os.Stderr, "ssg:", err)
		os.Exit(1)
	}
}
