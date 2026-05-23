// Package images handles responsive image resizing, WebP encoding, and
// an mtime-based cache.
package images

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// Pipeline resizes source images into multiple widths and writes them to
// outDir as both their original format and .webp.
type Pipeline struct {
	outDir string // absolute filesystem path, e.g. dist/_images
	outURL string // URL prefix, e.g. /_images
	widths []int

	mu    sync.Mutex
	cache map[string]*Variants // keyed by absolute source path
}

// Variants is the full set of resized outputs for one source image.
type Variants struct {
	SrcWidth, SrcHeight int
	Sizes               []Variant
}

// Variant is one rendered width.
type Variant struct {
	Width int
	URL   string // /_images/foo-abc12345-800.jpg
	WebP  string // /_images/foo-abc12345-800.webp
}

// New constructs a pipeline writing into outDir.
func New(outDir string, widths []int) *Pipeline {
	return &Pipeline{
		outDir: outDir,
		outURL: "/_images",
		widths: widths,
		cache:  map[string]*Variants{},
	}
}

// Process resizes srcPath into all configured widths.  Outputs already on
// disk are reused.  Safe to call concurrently from multiple goroutines.
func (p *Pipeline) Process(srcPath string) (*Variants, error) {
	srcPath, err := filepath.Abs(srcPath)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if v, ok := p.cache[srcPath]; ok {
		p.mu.Unlock()
		return v, nil
	}
	p.mu.Unlock()

	info, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", srcPath, err)
	}

	src, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", srcPath, err)
	}
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	if err := os.MkdirAll(p.outDir, 0o755); err != nil {
		return nil, err
	}

	hash := hashSrc(srcPath, info.ModTime().UnixNano())[:8]
	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		ext = ".jpg"
	}
	base := sanitize(strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath)))

	v := &Variants{SrcWidth: srcW, SrcHeight: srcH}

	wanted := chooseWidths(p.widths, srcW)
	for _, w := range wanted {
		name := fmt.Sprintf("%s-%s-%d", base, hash, w)
		jpgPath := filepath.Join(p.outDir, name+ext)
		webpPath := filepath.Join(p.outDir, name+".webp")

		if !fileExists(jpgPath) || !fileExists(webpPath) {
			// Resize once, encode twice.  When w equals the source width we
			// skip the resize step entirely to avoid any unnecessary sampling.
			var img = src
			if w != srcW {
				img = imaging.Resize(src, w, 0, imaging.Lanczos)
			}
			if !fileExists(jpgPath) {
				if err := imaging.Save(img, jpgPath, imaging.JPEGQuality(92)); err != nil {
					return nil, fmt.Errorf("save %s: %w", jpgPath, err)
				}
			}
			if !fileExists(webpPath) {
				f, err := os.Create(webpPath)
				if err != nil {
					return nil, err
				}
				if err := webp.Encode(f, img, &webp.Options{Quality: 88}); err != nil {
					f.Close()
					return nil, fmt.Errorf("encode %s: %w", webpPath, err)
				}
				f.Close()
			}
		}

		v.Sizes = append(v.Sizes, Variant{
			Width: w,
			URL:   p.outURL + "/" + name + ext,
			WebP:  p.outURL + "/" + name + ".webp",
		})
	}

	p.mu.Lock()
	p.cache[srcPath] = v
	p.mu.Unlock()
	return v, nil
}

// Largest returns the widest available variant.
func (v *Variants) Largest() Variant {
	if len(v.Sizes) == 0 {
		return Variant{}
	}
	return v.Sizes[len(v.Sizes)-1]
}

// Smallest returns the narrowest variant (used by thumbnails).
func (v *Variants) Smallest() Variant {
	if len(v.Sizes) == 0 {
		return Variant{}
	}
	return v.Sizes[0]
}

// SrcsetJPG renders a srcset attribute value for the original-format variants.
func (v *Variants) SrcsetJPG() string {
	parts := make([]string, 0, len(v.Sizes))
	for _, s := range v.Sizes {
		parts = append(parts, fmt.Sprintf("%s %dw", s.URL, s.Width))
	}
	return strings.Join(parts, ", ")
}

// SrcsetWebP renders a srcset attribute value for the WebP variants.
func (v *Variants) SrcsetWebP() string {
	parts := make([]string, 0, len(v.Sizes))
	for _, s := range v.Sizes {
		parts = append(parts, fmt.Sprintf("%s %dw", s.WebP, s.Width))
	}
	return strings.Join(parts, ", ")
}

// AspectHeight returns the proportional height for a given target width,
// useful as a CLS-prevention attribute on <img>.
func (v *Variants) AspectHeight(width int) int {
	if v.SrcWidth == 0 {
		return 0
	}
	return width * v.SrcHeight / v.SrcWidth
}

// maxVariantWidth caps the largest variant we'll ever emit, regardless of
// source size.  Above this, displays don't have enough device pixels to
// benefit and we'd just waste bytes (4K phone photos, etc.).
const maxVariantWidth = 2400

// chooseWidths returns the list of variant widths to emit for a source of
// width srcW.  Strategy:
//   - include each configured width strictly smaller than srcW
//   - append the source width as a non-upscaled variant for retina displays,
//     capped at maxVariantWidth
//
// Examples (widths = [400, 800, 1200, 1600]):
//
//	srcW=315           -> [315]
//	srcW=520           -> [400, 520]
//	srcW=1765          -> [400, 800, 1200, 1600, 1765]
//	srcW=6000 (capped) -> [400, 800, 1200, 1600, 2400]
func chooseWidths(widths []int, srcW int) []int {
	out := make([]int, 0, len(widths)+1)
	for _, w := range widths {
		if w < srcW {
			out = append(out, w)
		}
	}
	largest := srcW
	if largest > maxVariantWidth {
		largest = maxVariantWidth
	}
	// Avoid duplicating the largest configured width (e.g., srcW == 1600).
	if len(out) == 0 || largest > out[len(out)-1] {
		out = append(out, largest)
	}
	return out
}

func hashSrc(path string, mtime int64) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s|%d", path, mtime)
	return hex.EncodeToString(h.Sum(nil))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// sanitize keeps only [a-zA-Z0-9-_]; everything else becomes '-'.
func sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}
