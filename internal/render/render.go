// Package render parses theme templates and executes them with site data.
package render

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"

	"github.com/jbrodriguez/ssg/internal/content"
)

// SrcsetFunc emits an <img> or <picture> for a given image source.  Wired by
// the images package; falls back to a plain <img> if nil.
type SrcsetFunc func(src, alt, class string) template.HTML

// Renderer holds parsed templates and shared rendering state.
type Renderer struct {
	siteURL  string
	tmpl     *template.Template
	md       goldmark.Markdown
	srcsetFn SrcsetFunc
}

// New parses every *.html under templatesDir (recursively) into one template
// set so partials can be invoked via {{template "name" .}}.
func New(templatesDir, siteURL string) (*Renderer, error) {
	r := &Renderer{
		siteURL: siteURL,
		md:      newMarkdown(),
	}
	t := template.New("").Funcs(r.funcMap())
	if err := parseTree(t, templatesDir); err != nil {
		return nil, err
	}
	r.tmpl = t
	return r, nil
}

// SetSrcset wires the image helper.  Must be called after the images package
// finishes processing if you want responsive variants in output.
func (r *Renderer) SetSrcset(fn SrcsetFunc) { r.srcsetFn = fn }

// RenderMarkdown is exposed so the build orchestrator can convert post bodies
// before passing PageData to ExecuteToFile.
func (r *Renderer) RenderMarkdown(body []byte) (template.HTML, error) {
	return r.renderMarkdown(body)
}

// ExecuteToFile renders the named template to outPath, creating parent dirs.
func (r *Renderer) ExecuteToFile(name, outPath string, data any) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.Execute(f, name, data)
}

// Execute renders the named template to w.
func (r *Renderer) Execute(w io.Writer, name string, data any) error {
	if r.tmpl.Lookup(name) == nil {
		return fmt.Errorf("template %q not found", name)
	}
	return r.tmpl.ExecuteTemplate(w, name, data)
}

// parseTree walks dir and parses every *.html file into t.  Templates use
// {{define "name"}}...{{end}} blocks, so the file path is not the template
// name — the names come from the define directives inside.
func parseTree(t *template.Template, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := t.Parse(string(b)); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		return nil
	})
}

// Compile-time check: content.Post is a *Post in template funcs.
var _ = (*content.Post)(nil)
