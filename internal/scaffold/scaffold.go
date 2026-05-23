// Package scaffold creates new posts from a title.
package scaffold

import (
	"errors"

	"github.com/jbrodriguez/ssg/internal/config"
)

// NewPost is the entry point for `ssg new`.  Stub — implemented later.
func NewPost(cfg *config.Config, title string) error {
	_ = cfg
	_ = title
	return errors.New("ssg new: not yet implemented")
}
