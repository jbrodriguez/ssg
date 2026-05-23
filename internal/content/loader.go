package content

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/adrg/frontmatter"
)

// LoadPosts walks postsDir, parses each <slug>/index.md, filters drafts when
// filterDrafts is true, and returns posts sorted by date descending.
// Markdown bodies are NOT yet rendered to HTML here — that happens in render.
func LoadPosts(postsDir string, filterDrafts bool) ([]*Post, error) {
	entries, err := os.ReadDir(postsDir)
	if err != nil {
		return nil, fmt.Errorf("read posts dir %s: %w", postsDir, err)
	}

	var posts []*Post
	var drafts []string
	for _, e := range entries {
		if !e.IsDir() || isHidden(e.Name()) {
			continue
		}
		mdPath := filepath.Join(postsDir, e.Name(), "index.md")
		p, body, err := loadOne(mdPath, e.Name())
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("load %s: %w", mdPath, err)
		}
		_ = body // body is attached during render
		if filterDrafts && p.Status == "draft" {
			drafts = append(drafts, p.Slug)
			continue
		}
		posts = append(posts, p)
	}

	if len(drafts) > 0 {
		log.Printf("ssg: skipped %d draft posts: %v", len(drafts), drafts)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	return posts, nil
}

// loadOne parses a single post's frontmatter and returns its struct plus the
// raw markdown body (without frontmatter).
func loadOne(path, slug string) (*Post, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	var fm rawFrontmatter
	body, err := frontmatter.Parse(f, &fm)
	if err != nil {
		return nil, nil, err
	}

	return &Post{
		Slug:        slug,
		Path:        path,
		Title:       fm.Title,
		Subtitle:    fm.Subtitle,
		Description: fm.Description,
		Author:      fm.Author,
		Date:        fm.Date.Time,
		Updated:     fm.Updated.Time,
		Status:      fm.Status,
		Cover:       fm.Cover,
		Caption:     fm.Caption,
		Tags:        fm.Tags,
		Pixelfed:    fm.Pixelfed,
	}, body, nil
}

// Body re-reads the post's markdown body (without frontmatter).  Kept separate
// from LoadPosts so the loader stays cheap; render calls Body lazily.
func Body(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var fm rawFrontmatter
	return frontmatter.Parse(f, &fm)
}

func isHidden(name string) bool { return len(name) > 0 && name[0] == '.' }
