package render

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/jbrodriguez/ssg/internal/content"
	"github.com/jbrodriguez/ssg/internal/images"
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
		"cover":    r.cover,
		"thumb":    r.thumb,
		"now":      time.Now,
		"year":     func(t time.Time) int { return t.Year() },
	}
}

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

func isoDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func (r *Renderer) absURL(path string) string {
	return strings.TrimRight(r.siteURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func relURL(path string) string { return "/" + strings.TrimLeft(path, "/") }

func hasTag(p *content.Post, t string) bool { return p.HasTag(t) }

func safeHTML(s string) template.HTML { return template.HTML(s) }

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

// SetCovers wires the per-slug variants lookup used by the cover/thumb funcs.
func (r *Renderer) SetCovers(covers map[string]*images.Variants, fallback *images.Variants) {
	r.covers = covers
	r.fallbackCover = fallback
}

// cover emits a <picture>+<img> for a post's full-size cover.  Used by post.html.
// args: post, alt, class, sizes.
func (r *Renderer) cover(p *content.Post, alt, class, sizes string) template.HTML {
	v := r.lookupVariants(p)
	if v == nil {
		return template.HTML(fmt.Sprintf(`<img src="/static/default-post-header-img.jpg" alt=%q class=%q loading="eager">`, html.EscapeString(alt), html.EscapeString(class)))
	}
	return renderPicture(v, alt, class, sizes, "eager")
}

// thumb emits a smaller card image, used by blog_card.html.
// args: post, class.
func (r *Renderer) thumb(p *content.Post, class string) template.HTML {
	v := r.lookupVariants(p)
	if v == nil {
		return template.HTML(fmt.Sprintf(`<img src="/static/default-post-header-img.jpg" alt=%q class=%q loading="lazy" width="400" height="224">`, html.EscapeString(p.Title), html.EscapeString(class)))
	}
	return renderPicture(v, p.Title, class, "400px", "lazy")
}

func (r *Renderer) lookupVariants(p *content.Post) *images.Variants {
	if v, ok := r.covers[p.Slug]; ok {
		return v
	}
	return r.fallbackCover
}

func renderPicture(v *images.Variants, alt, class, sizes, loading string) template.HTML {
	largest := v.Largest()
	h := v.AspectHeight(largest.Width)
	return template.HTML(fmt.Sprintf(
		`<picture><source type="image/webp" srcset="%s" sizes="%s"><img src="%s" srcset="%s" sizes="%s" width="%d" height="%d" alt="%s" class="%s" loading="%s"></picture>`,
		v.SrcsetWebP(), sizes,
		largest.URL, v.SrcsetJPG(), sizes,
		largest.Width, h,
		html.EscapeString(alt), html.EscapeString(class), loading))
}
