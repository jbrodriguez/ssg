// Package config loads TOML configuration from ~/.config/ssg/<name>.toml
// and applies CLI flag overrides.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the resolved site configuration after TOML + flag overlay.
// All path fields are absolute by the time Load returns.
type Config struct {
	SiteURL         string `toml:"site_url"`
	SiteDomain      string `toml:"site_domain"`
	SiteTitle       string `toml:"site_title"`
	SiteAuthor      string `toml:"site_author"`
	SiteDescription string `toml:"site_description"`
	TwitterHandle   string `toml:"twitter_handle"`

	SiteRoot   string `toml:"site_root"`
	ContentDir string `toml:"content_dir"`
	ThemeDir   string `toml:"theme_dir"`
	PublicDir  string `toml:"public_dir"`
	DistDir    string `toml:"dist_dir"`

	PostsPerPage int   `toml:"posts_per_page"`
	ImageWidths  []int `toml:"image_widths"`
	FilterDrafts bool  `toml:"filter_drafts"`

	Socials []Social `toml:"socials"`
}

// Social is one entry in the site's social-links list.
type Social struct {
	Title string `toml:"title"`
	URL   string `toml:"url"`
}

// Overrides are CLI-flag values that take precedence over the TOML file.
// Empty/zero fields mean "no override".
type Overrides struct {
	SiteRoot string
}

// Load reads ~/.config/ssg/<name>.toml, applies overrides, fills in defaults,
// and resolves all directory paths to absolute paths under SiteRoot.
func Load(name string, ov Overrides) (*Config, error) {
	path, err := configPath(name)
	if err != nil {
		return nil, err
	}

	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	if ov.SiteRoot != "" {
		c.SiteRoot, _ = filepath.Abs(ov.SiteRoot)
	}
	if c.SiteRoot == "" {
		return nil, errors.New("site_root is required (set in config or pass --site-root)")
	}
	if !filepath.IsAbs(c.SiteRoot) {
		return nil, fmt.Errorf("site_root must be absolute: %s", c.SiteRoot)
	}

	c.ContentDir = resolve(c.SiteRoot, c.ContentDir, "data")
	c.ThemeDir = resolve(c.SiteRoot, c.ThemeDir, "theme")
	c.PublicDir = resolve(c.SiteRoot, c.PublicDir, "public")
	c.DistDir = resolve(c.SiteRoot, c.DistDir, "dist")

	if c.PostsPerPage <= 0 {
		c.PostsPerPage = 10
	}
	if len(c.ImageWidths) == 0 {
		c.ImageWidths = []int{400, 800, 1200, 1600}
	}

	if c.SiteURL == "" {
		return nil, errors.New("site_url is required")
	}

	return &c, nil
}

// PostsDir is content/posts.
func (c *Config) PostsDir() string { return filepath.Join(c.ContentDir, "posts") }

// AboutDir is content/about.
func (c *Config) AboutDir() string { return filepath.Join(c.ContentDir, "about") }

// UnbalancedDir is content/unbalanced.
func (c *Config) UnbalancedDir() string { return filepath.Join(c.ContentDir, "unbalanced") }

// TemplatesDir is theme/templates.
func (c *Config) TemplatesDir() string { return filepath.Join(c.ThemeDir, "templates") }

// CSSEntry is the source CSS file passed to tailwindcss.
func (c *Config) CSSEntry() string { return filepath.Join(c.ThemeDir, "assets", "css", "global.css") }

// StaticDir is theme/assets/static.
func (c *Config) StaticDir() string { return filepath.Join(c.ThemeDir, "assets", "static") }

// FontsDir is theme/assets/fonts.
func (c *Config) FontsDir() string { return filepath.Join(c.ThemeDir, "assets", "fonts") }

// ImagesOutDir is dist/_images.
func (c *Config) ImagesOutDir() string { return filepath.Join(c.DistDir, "_images") }

func resolve(root, val, fallback string) string {
	if val == "" {
		val = fallback
	}
	if filepath.IsAbs(val) {
		return val
	}
	return filepath.Join(root, val)
}

func configPath(name string) (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".toml"), nil
}

func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ssg"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ssg"), nil
}
