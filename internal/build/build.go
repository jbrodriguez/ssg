// Package build orchestrates the full site build.
package build

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jbrodriguez/ssg/internal/config"
	"github.com/jbrodriguez/ssg/internal/content"
	"github.com/jbrodriguez/ssg/internal/feed"
	"github.com/jbrodriguez/ssg/internal/images"
	"github.com/jbrodriguez/ssg/internal/render"
	"github.com/jbrodriguez/ssg/internal/serve"
	"github.com/jbrodriguez/ssg/internal/tailwind"
)

// Options controls runtime behavior of a build.
type Options struct {
	Serve bool
	Watch bool
	Port  int
}

// Run executes the build pipeline once, plus optionally serve and watch.
func Run(cfg *config.Config, opts Options) error {
	if err := buildOnce(cfg); err != nil {
		return err
	}
	if !opts.Serve {
		return nil
	}

	addr := fmt.Sprintf(":%d", opts.Port)
	if opts.Port == 0 {
		addr = ":4321"
	}
	srv := serve.New(cfg.DistDir, addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(ctx); err != nil {
			log.Printf("ssg: server: %v", err)
		}
	}()

	if opts.Watch {
		go func() {
			watchDirs := []string{cfg.ContentDir, cfg.ThemeDir, cfg.PublicDir}
			err := serve.Watch(watchDirs, 200*time.Millisecond, func() {
				log.Println("ssg: rebuilding (file change)")
				if err := buildOnce(cfg); err != nil {
					log.Printf("ssg: rebuild failed: %v", err)
					return
				}
				srv.Reload()
			})
			if err != nil {
				log.Printf("ssg: watcher: %v", err)
			}
		}()
	}

	wg.Wait()
	return nil
}

// buildOnce runs the full build pipeline a single time.
func buildOnce(cfg *config.Config) error {
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

	site := siteFromConfig(cfg)
	tags := content.TagsFromPosts(posts)
	byTag := content.PostsByTag(posts)

	if err := renderPosts(r, cfg, site, posts, byTag, covers); err != nil {
		return err
	}
	if err := renderHome(r, cfg, site, posts, tags); err != nil {
		return err
	}
	if err := renderPostsIndex(r, cfg, site, posts, tags); err != nil {
		return err
	}
	if err := renderTagPages(r, cfg, site, byTag, tags); err != nil {
		return err
	}
	if err := renderTagsIndex(r, cfg, site, posts, tags); err != nil {
		return err
	}
	if err := renderSingle(r, cfg, site, "unbalanced", "unbalanced/index.md", filepath.Join(cfg.DistDir, "unbalanced", "index.html"), "unbalanced :: "+site.Title); err != nil {
		log.Printf("ssg: unbalanced: %v", err)
	}
	if err := renderSingle(r, cfg, site, "about", "about/index.md", filepath.Join(cfg.DistDir, "about", "index.html"), "About :: "+site.Title); err != nil {
		log.Printf("ssg: about: %v (skipping)", err)
	}
	if err := render404(r, cfg, site); err != nil {
		return err
	}
	if err := writeRSS(cfg, site, posts, covers); err != nil {
		return err
	}
	if err := writeSitemap(cfg, site, posts, byTag); err != nil {
		return err
	}

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

func renderPosts(r *render.Renderer, cfg *config.Config, site *render.Site, posts []*content.Post, byTag map[string][]*content.Post, covers map[string]*images.Variants) error {
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
	return nil
}

func renderHome(r *render.Renderer, cfg *config.Config, site *render.Site, posts []*content.Post, tags []content.TagCount) error {
	latest := posts
	if len(latest) > cfg.PostsPerPage {
		latest = latest[:cfg.PostsPerPage]
	}
	topTags := tags
	if len(topTags) > 20 {
		topTags = topTags[:20]
	}
	data := render.PageData{
		Site:    site,
		Section: "",
		SEO:     seoForHome(site),
		Posts:   latest,
		Tags:    topTags,
	}
	out := filepath.Join(cfg.DistDir, "index.html")
	if err := r.ExecuteToFile("index", out, data); err != nil {
		return fmt.Errorf("render home: %w", err)
	}
	log.Printf("ssg: rendered home")
	return nil
}

func renderPostsIndex(r *render.Renderer, cfg *config.Config, site *render.Site, posts []*content.Post, tags []content.TagCount) error {
	per := cfg.PostsPerPage
	total := (len(posts) + per - 1) / per
	topTags := tags
	if len(topTags) > 20 {
		topTags = topTags[:20]
	}
	for i := 0; i < total; i++ {
		startIdx := i * per
		endIdx := startIdx + per
		if endIdx > len(posts) {
			endIdx = len(posts)
		}
		pageNum := i + 1
		var prev, next string
		if pageNum > 1 {
			if pageNum == 2 {
				prev = "/posts/"
			} else {
				prev = fmt.Sprintf("/posts/page/%d/", pageNum-1)
			}
		}
		if pageNum < total {
			next = fmt.Sprintf("/posts/page/%d/", pageNum+1)
		}
		data := render.PageData{
			Site:       site,
			Section:    "posts",
			SEO:        seoForPostsPage(site, pageNum, total),
			Posts:      posts[startIdx:endIdx],
			PageNum:    pageNum,
			TotalPages: total,
			PrevURL:    prev,
			NextURL:    next,
			Tags:       topTags,
		}
		var out string
		if pageNum == 1 {
			out = filepath.Join(cfg.DistDir, "posts", "index.html")
		} else {
			out = filepath.Join(cfg.DistDir, "posts", "page", fmt.Sprintf("%d", pageNum), "index.html")
		}
		if err := r.ExecuteToFile("posts_index", out, data); err != nil {
			return fmt.Errorf("render posts page %d: %w", pageNum, err)
		}
	}
	log.Printf("ssg: rendered %d posts-index pages", total)
	return nil
}

func renderTagPages(r *render.Renderer, cfg *config.Config, site *render.Site, byTag map[string][]*content.Post, allTags []content.TagCount) error {
	for tag, posts := range byTag {
		// "Other common tags" excludes the current one
		others := make([]content.TagCount, 0, len(allTags))
		for _, t := range allTags {
			if t.Name != tag {
				others = append(others, t)
			}
		}
		if len(others) > 20 {
			others = others[:20]
		}
		data := render.PageData{
			Site:       site,
			Section:    "posts",
			SEO:        seoForTag(site, tag),
			Posts:      posts,
			CurrentTag: tag,
			Tags:       others,
		}
		out := filepath.Join(cfg.DistDir, "tag", tag, "index.html")
		if err := r.ExecuteToFile("tag", out, data); err != nil {
			return fmt.Errorf("render tag %s: %w", tag, err)
		}
	}
	log.Printf("ssg: rendered %d tag pages", len(byTag))
	return nil
}

func renderTagsIndex(r *render.Renderer, cfg *config.Config, site *render.Site, posts []*content.Post, tags []content.TagCount) error {
	recent := posts
	if len(recent) > 4 {
		recent = recent[:4]
	}
	data := render.PageData{
		Site:    site,
		Section: "",
		SEO:     seoForTagsIndex(site),
		Posts:   recent,
		Tags:    tags,
	}
	out := filepath.Join(cfg.DistDir, "tag", "index.html")
	if err := r.ExecuteToFile("tags_index", out, data); err != nil {
		return fmt.Errorf("render tags index: %w", err)
	}
	log.Printf("ssg: rendered tags index")
	return nil
}

func renderSingle(r *render.Renderer, cfg *config.Config, site *render.Site, section, relPath, outPath, title string) error {
	srcPath := filepath.Join(cfg.ContentDir, relPath)
	page, err := content.LoadSingle(srcPath)
	if err != nil {
		return err
	}
	html, err := r.RenderMarkdown(page.Body)
	if err != nil {
		return err
	}
	data := render.PageData{
		Site:     site,
		Section:  section,
		SEO:      seoForSingle(site, title, page.Title),
		BodyHTML: html,
	}
	if err := r.ExecuteToFile("markdown_page", outPath, data); err != nil {
		return fmt.Errorf("render %s: %w", section, err)
	}
	// Copy any sibling images so relative <img src="./foo.jpg"> in the
	// markdown body resolves from the output directory.
	if err := copySiblingAssets(filepath.Dir(srcPath), filepath.Dir(outPath)); err != nil {
		log.Printf("ssg: copy %s assets: %v", section, err)
	}
	log.Printf("ssg: rendered %s", section)
	return nil
}

// copySiblingAssets copies image-like files from srcDir to dstDir, skipping
// the index markdown sources themselves.
func copySiblingAssets(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg":
		default:
			continue
		}
		if err := copyFile(filepath.Join(srcDir, name), filepath.Join(dstDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func writeRSS(cfg *config.Config, site *render.Site, posts []*content.Post, covers map[string]*images.Variants) error {
	f := &feed.Feed{
		Title:       site.Title,
		Description: "Recent content in Posts on " + site.Title,
		Link:        site.URL + "/posts",
		SelfLink:    site.URL + "/posts/rss.xml",
		Language:    "en-us",
		LastBuild:   time.Now(),
	}
	for _, p := range posts {
		var imgTag string
		if v := covers[p.Slug]; v != nil && len(v.Sizes) > 0 {
			largest := v.Largest()
			imgTag = fmt.Sprintf(
				`<img src="%s%s" alt="%s feature image" width="%d" height="%d" loading="lazy" decoding="async"/>`,
				site.URL, largest.URL, p.Title, largest.Width, v.AspectHeight(largest.Width))
		}
		f.Items = append(f.Items, feed.Item{
			Title:       p.Title,
			Link:        p.AbsURL(site.URL),
			PubDate:     p.Date,
			Description: p.Description,
			Content:     imgTag + string(p.HTML),
		})
	}
	out := filepath.Join(cfg.DistDir, "posts", "rss.xml")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	fout, err := os.Create(out)
	if err != nil {
		return err
	}
	defer fout.Close()
	if err := f.WriteRSS(fout); err != nil {
		return fmt.Errorf("rss: %w", err)
	}
	log.Printf("ssg: wrote rss.xml (%d items)", len(f.Items))
	return nil
}

func writeSitemap(cfg *config.Config, site *render.Site, posts []*content.Post, byTag map[string][]*content.Post) error {
	urls := []feed.SitemapURL{
		{Loc: site.URL + "/"},
		{Loc: site.URL + "/posts/"},
		{Loc: site.URL + "/about/"},
		{Loc: site.URL + "/unbalanced/"},
		{Loc: site.URL + "/tag/"},
	}
	for _, p := range posts {
		urls = append(urls, feed.SitemapURL{Loc: p.AbsURL(site.URL), LastMod: p.Date})
	}
	for tag := range byTag {
		urls = append(urls, feed.SitemapURL{Loc: site.URL + "/tag/" + tag + "/"})
	}

	out0 := filepath.Join(cfg.DistDir, "sitemap-0.xml")
	f0, err := os.Create(out0)
	if err != nil {
		return err
	}
	if err := feed.WriteSitemap(f0, urls); err != nil {
		f0.Close()
		return fmt.Errorf("sitemap-0: %w", err)
	}
	f0.Close()

	outIdx := filepath.Join(cfg.DistDir, "sitemap-index.xml")
	fIdx, err := os.Create(outIdx)
	if err != nil {
		return err
	}
	if err := feed.WriteSitemapIndex(fIdx, []string{site.URL + "/sitemap-0.xml"}); err != nil {
		fIdx.Close()
		return fmt.Errorf("sitemap-index: %w", err)
	}
	fIdx.Close()
	log.Printf("ssg: wrote sitemap (%d urls)", len(urls))
	return nil
}

func render404(r *render.Renderer, cfg *config.Config, site *render.Site) error {
	data := render.PageData{
		Site:    site,
		Section: "",
		SEO:     seoForSingle(site, "404 :: "+site.Title, "Not found"),
	}
	out := filepath.Join(cfg.DistDir, "404.html")
	if err := r.ExecuteToFile("notfound", out, data); err != nil {
		return fmt.Errorf("render 404: %w", err)
	}
	log.Printf("ssg: rendered 404")
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

// SEO helpers --------------------------------------------------------------

func defaultSEO(site *render.Site) render.SEO {
	return render.SEO{
		Title:           site.Title,
		Description:     site.Description,
		Image:           site.URL + "/static/jb.png",
		ImageWidth:      512,
		ImageHeight:     512,
		ImageAlt:        "Cover picture for " + site.Title,
		Canonical:       site.URL + "/",
		OGType:          "website",
		Generator:       "ssg",
		Sitemap:         site.URL + "/sitemap-index.xml",
		RSSURL:          site.URL + "/posts/rss.xml",
		RSSTitle:        site.Title,
		TwitterHandle:   site.Twitter,
		TwitterCardType: "summary_large_image",
	}
}

func seoForHome(site *render.Site) render.SEO {
	s := defaultSEO(site)
	s.Description = "The website of " + site.Author + ": " + site.Description
	return s
}

func seoForPost(site *render.Site, p *content.Post) render.SEO {
	s := defaultSEO(site)
	s.Title = fmt.Sprintf("%s :: %s", p.Title, site.Title)
	if p.Description != "" {
		s.Description = p.Description
	} else {
		s.Description = "A post published by " + site.Title
	}
	s.OGType = "article"
	s.Canonical = p.AbsURL(site.URL)
	return s
}

func seoForPostsPage(site *render.Site, pageNum, total int) render.SEO {
	s := defaultSEO(site)
	s.Title = fmt.Sprintf("%s's Blog - Page %d", site.Title, pageNum)
	s.Description = fmt.Sprintf("Page %d of %d of %s's blog. Here you will find all the articles published by %s in the last years.", pageNum, total, site.Title, site.Title)
	if pageNum == 1 {
		s.Canonical = site.URL + "/posts/"
	} else {
		s.Canonical = fmt.Sprintf("%s/posts/page/%d/", site.URL, pageNum)
	}
	return s
}

func seoForTag(site *render.Site, tag string) render.SEO {
	s := defaultSEO(site)
	s.Title = fmt.Sprintf("Tags :: %s", site.Title)
	s.Description = fmt.Sprintf("%s's posts under the tag %q.", site.Title, tag)
	s.Canonical = site.URL + "/tag/" + tag + "/"
	return s
}

func seoForTagsIndex(site *render.Site) render.SEO {
	s := defaultSEO(site)
	s.Title = fmt.Sprintf("Tags :: %s", site.Title)
	s.Description = fmt.Sprintf("Page containing the list of tags and recent posts from %s.", site.Title)
	s.Canonical = site.URL + "/tag/"
	return s
}

func seoForSingle(site *render.Site, title, _ string) render.SEO {
	s := defaultSEO(site)
	s.Title = title
	return s
}

// File copy ----------------------------------------------------------------

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
