package content

import (
	"os"

	"github.com/adrg/frontmatter"
)

// SinglePage is a non-post markdown page (about, unbalanced, etc.).
type SinglePage struct {
	Title  string
	Author string
	Date   flexibleDate
	Body   []byte // raw markdown body, ready for goldmark
}

// rawSingle is the frontmatter shape for non-post pages.
type rawSingle struct {
	Title  string       `yaml:"title"`
	Author string       `yaml:"author"`
	Date   flexibleDate `yaml:"date"`
}

// LoadSingle parses a single markdown file (frontmatter + body).
// Used by about, unbalanced, and similar one-off pages.
func LoadSingle(path string) (*SinglePage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var fm rawSingle
	body, err := frontmatter.Parse(f, &fm)
	if err != nil {
		return nil, err
	}
	return &SinglePage{
		Title:  fm.Title,
		Author: fm.Author,
		Date:   fm.Date,
		Body:   body,
	}, nil
}
