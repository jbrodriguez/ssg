package serve

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch recursively watches the given directories and calls onChange after
// a short debounce window whenever any non-ignored file changes.  Blocks
// until the underlying watcher errors out.  Returns the first error.
func Watch(dirs []string, debounce time.Duration, onChange func()) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	for _, d := range dirs {
		if err := addRecursive(w, d); err != nil {
			return err
		}
	}

	var (
		timer    *time.Timer
		fireOnce = func() {
			if onChange != nil {
				onChange()
			}
		}
	)

	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if shouldIgnore(ev.Name) {
				continue
			}
			// If a new directory was created, start watching it too.
			if ev.Has(fsnotify.Create) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = addRecursive(w, ev.Name)
				}
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, fireOnce)
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("ssg: watcher: %v", err)
		}
	}
}

func addRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if shouldIgnore(path) {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}

// shouldIgnore filters out paths we don't care about: VCS dirs, output dirs,
// node_modules, editor swap files.
func shouldIgnore(path string) bool {
	base := filepath.Base(path)
	switch base {
	case ".git", "node_modules", "dist", ".astro":
		return true
	}
	if strings.HasSuffix(base, "~") || strings.HasSuffix(base, ".swp") || strings.HasPrefix(base, ".#") {
		return true
	}
	return false
}
