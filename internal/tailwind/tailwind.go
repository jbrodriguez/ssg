// Package tailwind shells out to the tailwindcss binary.
package tailwind

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrMissing is returned by Check when tailwindcss is not found.
var ErrMissing = errors.New("tailwindcss binary not found — install via 'brew install tailwindcss'")

// preferredPaths are checked before exec.LookPath so that a stale rbenv/npm
// shim doesn't shadow the Homebrew install.
var preferredPaths = []string{
	"/opt/homebrew/bin/tailwindcss", // Apple Silicon Homebrew
	"/usr/local/bin/tailwindcss",    // Intel Homebrew / manual install
}

// Check returns the path to a tailwindcss binary, preferring Homebrew over
// whatever happens to be on PATH (since rbenv shims often shadow it).
// The TAILWINDCSS env var, if set, takes precedence over both.
func Check() (string, error) {
	if v := os.Getenv("TAILWINDCSS"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
	}
	for _, p := range preferredPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if path, err := exec.LookPath("tailwindcss"); err == nil {
		return path, nil
	}
	return "", ErrMissing
}

// Build runs `tailwindcss -i <in> -o <out> --minify`.
func Build(in, out string) error {
	bin, err := Check()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "-i", in, "-o", out, "--minify")
	cmd.Dir = parentDir(in)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tailwindcss: %w\n%s", err, output)
	}
	return nil
}

func parentDir(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "."
	}
	return path[:i]
}
