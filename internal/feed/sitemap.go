package feed

import (
	"encoding/xml"
	"io"
	"time"
)

const sitemapNS = "http://www.sitemaps.org/schemas/sitemap/0.9"

// SitemapURL is one entry in the urlset.
type SitemapURL struct {
	Loc        string
	LastMod    time.Time
	ChangeFreq string
}

// WriteSitemapIndex writes a <sitemapindex> referencing child sitemaps.
func WriteSitemapIndex(w io.Writer, childLocs []string) error {
	doc := sitemapIndexDoc{XMLNS: sitemapNS}
	for _, loc := range childLocs {
		doc.Sitemaps = append(doc.Sitemaps, sitemapEntry{Loc: loc})
	}
	return writeXML(w, doc)
}

// WriteSitemap writes a <urlset> containing the given URLs.
func WriteSitemap(w io.Writer, urls []SitemapURL) error {
	doc := sitemapDoc{XMLNS: sitemapNS}
	for _, u := range urls {
		e := urlEntry{Loc: u.Loc, ChangeFreq: u.ChangeFreq}
		if !u.LastMod.IsZero() {
			e.LastMod = u.LastMod.UTC().Format("2006-01-02")
		}
		doc.URLs = append(doc.URLs, e)
	}
	return writeXML(w, doc)
}

type sitemapIndexDoc struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	XMLNS    string         `xml:"xmlns,attr"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc string `xml:"loc"`
}

type sitemapDoc struct {
	XMLName xml.Name   `xml:"urlset"`
	XMLNS   string     `xml:"xmlns,attr"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
}

func writeXML(w io.Writer, doc any) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}
