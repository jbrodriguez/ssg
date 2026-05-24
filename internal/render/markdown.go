package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// WriteChromaCSS emits two chroma stylesheets — one scoped to
// [data-theme="light"], one scoped to [data-theme="dark"] — using the styles
// configured on the renderer.  Call after tailwind compiles and append to
// the main stylesheet.  Chroma class names are theme-independent so the
// HTML output doesn't depend on which styles are chosen here.
func (r *Renderer) WriteChromaCSS(w io.Writer) error {
	if err := writeScopedChromaCSS(w, r.chromaLight, `[data-theme="light"] `); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return writeScopedChromaCSS(w, r.chromaDark, `[data-theme="dark"] `)
}

// writeScopedChromaCSS writes the chroma stylesheet for styleName with every
// `.chroma` selector prefixed by `prefix`, so the rules only apply when the
// page has that data-theme.
func writeScopedChromaCSS(w io.Writer, styleName, prefix string) error {
	style := styles.Get(styleName)
	if style == nil {
		return fmt.Errorf("chroma style %q not found", styleName)
	}
	var buf bytes.Buffer
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	if err := formatter.WriteCSS(&buf, style); err != nil {
		return err
	}
	scoped := strings.ReplaceAll(buf.String(), ".chroma", prefix+".chroma")
	_, err := io.WriteString(w, scoped)
	return err
}

// newMarkdown constructs the shared goldmark instance.  Configured for:
//   - GFM (tables, strikethrough, autolinks, task lists)
//   - footnotes
//   - chroma syntax highlighting (renderStyle picks one for class generation;
//     the actual colors come from the CSS emitted by WriteChromaCSS)
//   - unsafe HTML passthrough (post bodies contain inline HTML)
//   - autolinks + heading IDs
func newMarkdown(renderStyle string) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Table,
			highlighting.NewHighlighting(
				highlighting.WithStyle(renderStyle),
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
