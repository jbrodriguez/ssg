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
		"cover":     r.cover,
		"thumb":     r.thumb,
		"thumbLazy": r.thumbLazy,
		"card":      card,
		"cardLazy":  cardLazy,
		"dict":     dict,
		"now":      time.Now,
		"year":     func(t time.Time) int { return t.Year() },
		"sub":      func(a, b int) int { return a - b },
		"add":      func(a, b int) int { return a + b },
		"seq":      seq,
	}
}

// dict builds a map[string]any from key/value pairs.  Useful for passing
// inline structured data to a partial template, e.g.
// `{{template "x" (dict "Title" "hi" "Count" 3)}}`.
func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of args")
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		k, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at %d not a string", i)
		}
		m[k] = pairs[i+1]
	}
	return m, nil
}

// card builds a Card wrapper used by the blog_card partial.
func card(p *content.Post, big bool) Card { return Card{Post: p, Big: big} }

// cardLazy builds a Card whose thumbnail will be lazy-loaded.  Use for
// cards that render well below the fold (e.g. similar-posts on post.html).
func cardLazy(p *content.Post, big bool) Card { return Card{Post: p, Big: big, Lazy: true} }

// seq returns [start, start+1, ..., end] inclusive.  Used by pagination
// templates that need to iterate over page numbers.
func seq(start, end int) []int {
	if end < start {
		return nil
	}
	out := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		out = append(out, i)
	}
	return out
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
// Loading attribute is omitted (defaults to eager) since hero covers are
// always above the fold.
// args: post, alt, class, sizes.
func (r *Renderer) cover(p *content.Post, alt, class, sizes string) template.HTML {
	v := r.lookupVariants(p)
	if v == nil {
		return template.HTML(fmt.Sprintf(`<img src="/static/default-post-header-img.jpg" alt=%q class=%q>`, html.EscapeString(alt), html.EscapeString(class)))
	}
	return renderPicture(v, alt, class, sizes, "")
}

// thumb emits a card thumbnail with default (eager) loading.  Use for cards
// on listing pages (home, /posts/, /tag/).
// args: post, class.
func (r *Renderer) thumb(p *content.Post, class string) template.HTML {
	return r.thumbWith(p, class, "")
}

// thumbLazy emits a lazy-loaded card thumbnail.  Use for similar-posts and
// other cards that sit well below the fold.
func (r *Renderer) thumbLazy(p *content.Post, class string) template.HTML {
	return r.thumbWith(p, class, "lazy")
}

func (r *Renderer) thumbWith(p *content.Post, class, loading string) template.HTML {
	v := r.lookupVariants(p)
	if v == nil {
		loadAttr := ""
		if loading != "" {
			loadAttr = fmt.Sprintf(` loading=%q`, loading)
		}
		return template.HTML(fmt.Sprintf(`<img src="/static/default-post-header-img.jpg" alt=%q class=%q width="400" height="224"%s>`, html.EscapeString(p.Title), html.EscapeString(class), loadAttr))
	}
	return renderPicture(v, p.Title, class, "400px", loading)
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
	loadAttr := ""
	if loading != "" {
		loadAttr = fmt.Sprintf(` loading=%q`, loading)
	}
	return template.HTML(fmt.Sprintf(
		`<picture><source type="image/webp" srcset="%s" sizes="%s"><img src="%s" srcset="%s" sizes="%s" width="%d" height="%d" alt="%s" class="%s"%s></picture>`,
		v.SrcsetWebP(), sizes,
		largest.URL, v.SrcsetJPG(), sizes,
		largest.Width, h,
		html.EscapeString(alt), html.EscapeString(class), loadAttr))
}
