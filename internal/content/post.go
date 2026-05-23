// Package content loads markdown posts and computes derived data
// (tag indices, similar-posts).
package content

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

// Post is one rendered post ready for template execution.
type Post struct {
	// Identity
	Slug string // dir name under content/posts, used in URL
	Path string // path to the source index.md

	// Frontmatter
	Title       string
	Subtitle    string
	Description string
	Author      string
	Date        time.Time
	Updated     time.Time
	Status      string
	Cover       string // relative path to cover image (resolved during render)
	Caption     string
	Tags        []string
	Pixelfed    string

	// Rendered
	HTML template.HTML // markdown body rendered to HTML
}

// URL returns the post's site-relative URL (e.g. /posts/foo/).
func (p *Post) URL() string { return "/posts/" + p.Slug + "/" }

// AbsURL returns the absolute URL given a base site URL (no trailing slash).
func (p *Post) AbsURL(siteURL string) string {
	return strings.TrimRight(siteURL, "/") + p.URL()
}

// HasTag reports whether the post is tagged with t.
func (p *Post) HasTag(t string) bool {
	for _, tag := range p.Tags {
		if tag == t {
			return true
		}
	}
	return false
}

// rawFrontmatter is the unmarshal target for the YAML block at the top of
// each post.  It uses flexibleDate so both bare YYYY-MM-DD and RFC3339 parse.
type rawFrontmatter struct {
	Title       string       `yaml:"title"`
	Subtitle    string       `yaml:"subtitle"`
	Description string       `yaml:"description"`
	Author      string       `yaml:"author"`
	Date        flexibleDate `yaml:"date"`
	Updated     flexibleDate `yaml:"updated"`
	Status      string       `yaml:"status"`
	Cover       string       `yaml:"cover"`
	Caption     string       `yaml:"caption"`
	Tags        []string     `yaml:"tags"`
	Pixelfed    string       `yaml:"pixelfed"`
}

// flexibleDate accepts either a YAML date (parsed as time.Time by yaml.v3) or
// a YAML string in YYYY-MM-DD / RFC3339 form.
type flexibleDate struct {
	time.Time
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *flexibleDate) UnmarshalYAML(unmarshal func(any) error) error {
	var t time.Time
	if err := unmarshal(&t); err == nil {
		d.Time = t
		return nil
	}
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if parsed, err := time.Parse(layout, s); err == nil {
			d.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("date %q does not match supported layouts", s)
}
