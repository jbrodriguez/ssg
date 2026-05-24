package cli

import (
	"fmt"

	"github.com/jbrodriguez/ssg/internal/config"
)

// InitCmd is `ssg init`.  Creates a default ssg.toml in the current
// directory so a freshly-cloned site has something to start from.
type InitCmd struct{}

// Run executes the init subcommand.
func (i *InitCmd) Run(_ *Root) error {
	if err := config.Init(); err != nil {
		return err
	}
	fmt.Println("ssg: created ssg.toml — edit it and run `ssg build`")
	return nil
}
