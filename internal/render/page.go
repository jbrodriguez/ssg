package render

import (
	"github.com/jbrodriguez/ssg/internal/content"
	"github.com/jbrodriguez/ssg/internal/images"
)

// Site holds the site-wide constants surfaced to templates.
type Site struct {
	URL         string
	Domain      string
	Title       string
	Author      string
	Description string
	Twitter     string
	Socials     []Social
}

// Social is one entry in the site's social-links list.
type Social struct {
	Title string
	URL   string
}

// SEO is the set of head-tag values used by the "seo" partial.
type SEO struct {
	Title           string
	Description     string
	Image           string
	ImageWidth      int
	ImageHeight     int
	ImageAlt        string
	Canonical       string
	OGType          string // "website" or "article"
	Generator       string
	Sitemap         string
	RSSURL          string
	RSSTitle        string
	TwitterHandle   string
	TwitterCardType string
}

// Card wraps a Post with display hints, used by the blog_card partial.
// Lazy controls whether the card's thumbnail is loaded eagerly (listings,
// typically above the fold) or lazily (similar-posts at the bottom of a
// post page, well below the fold).
type Card struct {
	Post *content.Post
	Big  bool
	Lazy bool
}

// PageData is the data shape passed to every page template.
// Page-specific fields below are only populated for the relevant page.
type PageData struct {
	Site    *Site
	Section string // "posts", "about", or ""
	SEO     SEO

	// Post page
	Post    *content.Post
	Cover   *images.Variants
	Similar []*content.Post

	// Index / pagination / tag pages
	Posts      []*content.Post
	PageNum    int
	TotalPages int
	PrevURL    string
	NextURL    string
	CurrentTag string
	Tags       []content.TagCount

	// About / unbalanced pages (pre-rendered markdown bodies)
	BodyHTML interface{} // template.HTML
}
