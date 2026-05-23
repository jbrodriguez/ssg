// Package tailwind shells out to the tailwindcss binary.
package tailwind

import (
	"errors"
	"fmt"
	"os/exec"
)

// ErrMissing is returned by Check when tailwindcss is not on PATH.
var ErrMissing = errors.New("tailwindcss binary not found on PATH — install via 'brew install tailwindcss'")

// Check verifies the tailwindcss binary is available.
func Check() (string, error) {
	path, err := exec.LookPath("tailwindcss")
	if err != nil {
		return "", ErrMissing
	}
	return path, nil
}

// Build runs `tailwindcss -i <in> -o <out> --minify`.
func Build(in, out string) error {
	bin, err := Check()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "-i", in, "-o", out, "--minify")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tailwindcss: %w\n%s", err, output)
	}
	return nil
}
