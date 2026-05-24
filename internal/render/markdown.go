package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// chromaStyleName is the syntax-highlight palette chroma uses.  Kept in sync
// with the WriteChromaCSS output so the CSS classes match the spans.
const chromaStyleName = "github-dark"

// WriteChromaCSS dumps the chroma stylesheet for chromaStyleName, scoped to
// .chroma so it doesn't bleed into other elements.  Call this after tailwind
// finishes and append to the main stylesheet.
func WriteChromaCSS(w io.Writer) error {
	style := styles.Get(chromaStyleName)
	if style == nil {
		return fmt.Errorf("chroma style %q not found", chromaStyleName)
	}
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	return formatter.WriteCSS(w, style)
}

// newMarkdown constructs the shared goldmark instance.  Configured for:
//   - GFM (tables, strikethrough, autolinks, task lists)
//   - footnotes
//   - chroma syntax highlighting
//   - unsafe HTML passthrough (post bodies contain inline HTML)
//   - autolinks + heading IDs
func newMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Table,
			highlighting.NewHighlighting(
				highlighting.WithStyle(chromaStyleName),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

// renderMarkdown converts a markdown body to HTML.
func (r *Renderer) renderMarkdown(body []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(body, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
