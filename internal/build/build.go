// Package build orchestrates the full site build.
package build

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jbrodriguez/ssg/internal/config"
	"github.com/jbrodriguez/ssg/internal/content"
	"github.com/jbrodriguez/ssg/internal/images"
	"github.com/jbrodriguez/ssg/internal/render"
	"github.com/jbrodriguez/ssg/internal/tailwind"
)

// Options controls runtime behavior of a build.
type Options struct {
	Serve bool
	Watch bool
	Port  int
}

// Run executes the build pipeline.
func Run(cfg *config.Config, opts Options) error {
	start := time.Now()

	if _, err := tailwind.Check(); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.DistDir, 0o755); err != nil {
		return err
	}

	posts, err := content.LoadPosts(cfg.PostsDir(), cfg.FilterDrafts)
	if err != nil {
		return err
	}
	log.Printf("ssg: loaded %d posts", len(posts))

	r, err := render.New(cfg.TemplatesDir(), cfg.SiteURL)
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	// Render markdown bodies up-front.
	for _, p := range posts {
		body, err := content.Body(p.Path)
		if err != nil {
			return fmt.Errorf("read body %s: %w", p.Path, err)
		}
		html, err := r.RenderMarkdown(body)
		if err != nil {
			return fmt.Errorf("render markdown %s: %w", p.Path, err)
		}
		p.HTML = html
	}

	// Image pipeline: process every post cover plus the site-wide default.
	pipeline := images.New(cfg.ImagesOutDir(), cfg.ImageWidths)
	covers := map[string]*images.Variants{}
	for _, p := range posts {
		if p.Cover == "" {
			continue
		}
		coverSrc := filepath.Join(filepath.Dir(p.Path), p.Cover)
		v, err := pipeline.Process(coverSrc)
		if err != nil {
			log.Printf("ssg: cover for %s (%s): %v", p.Slug, p.Cover, err)
			continue
		}
		covers[p.Slug] = v
	}

	defaultCoverSrc := filepath.Join(cfg.StaticDir(), "default-post-header-img.jpg")
	fallback, err := pipeline.Process(defaultCoverSrc)
	if err != nil {
		log.Printf("ssg: default cover: %v", err)
	}
	r.SetCovers(covers, fallback)
	log.Printf("ssg: processed %d covers (+ default)", len(covers))

	// Per-post HTML
	site := siteFromConfig(cfg)
	byTag := content.PostsByTag(posts)
	for _, p := range posts {
		similar := content.SimilarPosts(p, byTag, 6)
		data := render.PageData{
			Site:    site,
			Section: "posts",
			SEO:     seoForPost(site, p),
			Post:    p,
			Cover:   covers[p.Slug],
			Similar: similar,
		}
		out := filepath.Join(cfg.DistDir, "posts", p.Slug, "index.html")
		if err := r.ExecuteToFile("post", out, data); err != nil {
			return fmt.Errorf("render post %s: %w", p.Slug, err)
		}
	}
	log.Printf("ssg: rendered %d post pages", len(posts))

	// CSS
	cssOut := filepath.Join(cfg.DistDir, "styles.css")
	if err := tailwind.Build(cfg.CSSEntry(), cssOut); err != nil {
		return fmt.Errorf("tailwind: %w", err)
	}
	log.Printf("ssg: built %s", cssOut)

	// Static assets and fonts
	if err := copyDir(cfg.StaticDir(), filepath.Join(cfg.DistDir, "static")); err != nil {
		return fmt.Errorf("copy static: %w", err)
	}
	if err := copyDir(cfg.FontsDir(), filepath.Join(cfg.DistDir, "fonts")); err != nil {
		return fmt.Errorf("copy fonts: %w", err)
	}
	if err := copyDir(cfg.PublicDir, cfg.DistDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("copy public: %w", err)
	}

	log.Printf("ssg: build complete in %s", time.Since(start).Round(time.Millisecond))
	return nil
}

func siteFromConfig(cfg *config.Config) *render.Site {
	socials := make([]render.Social, len(cfg.Socials))
	for i, s := range cfg.Socials {
		socials[i] = render.Social{Title: s.Title, URL: s.URL}
	}
	return &render.Site{
		URL:         cfg.SiteURL,
		Domain:      cfg.SiteDomain,
		Title:       cfg.SiteTitle,
		Author:      cfg.SiteAuthor,
		Description: cfg.SiteDescription,
		Twitter:     cfg.TwitterHandle,
		Socials:     socials,
	}
}

func seoForPost(site *render.Site, p *content.Post) render.SEO {
	desc := p.Description
	if desc == "" {
		desc = "A post published by " + site.Title
	}
	return render.SEO{
		Title:           fmt.Sprintf("%s :: %s", p.Title, site.Title),
		Description:     desc,
		Image:           site.URL + "/static/jb.png",
		ImageWidth:      512,
		ImageHeight:     512,
		ImageAlt:        "Cover picture for " + site.Title,
		Canonical:       p.AbsURL(site.URL),
		OGType:          "article",
		Generator:       "ssg",
		Sitemap:         site.URL + "/sitemap-index.xml",
		RSSURL:          site.URL + "/posts/rss.xml",
		RSSTitle:        site.Title,
		TwitterHandle:   site.Twitter,
		TwitterCardType: "summary_large_image",
	}
}

// copyDir mirrors srcDir into dstDir recursively.
func copyDir(srcDir, dstDir string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", srcDir)
	}
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dstDir, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		return copyFile(path, dst)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
