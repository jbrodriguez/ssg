package render

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/jbrodriguez/ssg/internal/content"
)

// funcMap returns the template func map.  All site-wide helpers used by
// templates live here.
func (r *Renderer) funcMap() template.FuncMap {
	return template.FuncMap{
		"fmtDate":  fmtDate,
		"isoDate":  isoDate,
		"shareURL": r.shareURL,
		"absURL":   r.absURL,
		"relURL":   relURL,
		"hasTag":   hasTag,
		"safeHTML": safeHTML,
		"srcset":   r.srcset, // image helper, stubbed until images pkg lands
		"now":      time.Now,
		"year":     func(t time.Time) int { return t.Year() },
	}
}

// fmtDate formats a time as "2006-01-02 15:04".
func fmtDate(t time.Time, layout ...string) string {
	if t.IsZero() {
		return ""
	}
	l := "2006-01-02 15:04"
	if len(layout) > 0 && layout[0] != "" {
		l = layout[0]
	}
	return t.Format(l)
}

// isoDate emits an ISO 8601 (RFC 3339) date string.
func isoDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// absURL returns an absolute URL by joining the site URL with a path.
func (r *Renderer) absURL(path string) string {
	return strings.TrimRight(r.siteURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// relURL ensures a site-relative URL begins with a single slash.
func relURL(path string) string {
	return "/" + strings.TrimLeft(path, "/")
}

// hasTag reports whether a post has tag t.
func hasTag(p *content.Post, t string) bool { return p.HasTag(t) }

// safeHTML marks an interpolated string as already HTML-safe.
func safeHTML(s string) template.HTML { return template.HTML(s) }

// shareURL builds a share URL for the given post and platform.
func (r *Renderer) shareURL(p *content.Post, platform string) string {
	abs := p.AbsURL(r.siteURL)
	switch platform {
	case "twitter", "x":
		return "https://twitter.com/intent/tweet?text=" +
			url.QueryEscape(p.Title) + "&url=" + url.QueryEscape(abs)
	case "facebook":
		return "https://www.facebook.com/sharer/sharer.php?u=" + url.QueryEscape(abs)
	case "linkedin":
		return "https://www.linkedin.com/sharing/share-offsite/?mini=true&url=" +
			url.QueryEscape(abs) + "&title=" + url.QueryEscape(p.Title)
	}
	return abs
}

// srcset emits an <img> with width-based srcset.  Stubbed: returns a plain
// <img> until the images package is wired.  Real impl lives in internal/images
// and is registered via SetSrcset at construction time.
func (r *Renderer) srcset(src, alt string, classes ...string) template.HTML {
	cls := strings.Join(classes, " ")
	if r.srcsetFn != nil {
		return r.srcsetFn(src, alt, cls)
	}
	return template.HTML(fmt.Sprintf(`<img src=%q alt=%q class=%q loading="lazy">`, src, alt, cls))
}
