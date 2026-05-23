// Package feed emits RSS and sitemap XML.
package feed

import (
	"encoding/xml"
	"io"
	"time"
)

const (
	rssVersion = "2.0"
	nsAtom     = "http://www.w3.org/2005/Atom"
	nsContent  = "http://purl.org/rss/1.0/modules/content/"
)

// Feed is the public RSS shape consumed by build.go.
type Feed struct {
	Title       string
	Description string
	Link        string // e.g. https://jbrio.net/posts
	SelfLink    string // e.g. https://jbrio.net/posts/rss.xml
	Language    string
	LastBuild   time.Time
	Items       []Item
}

// Item is one entry in the feed.
type Item struct {
	Title       string
	Link        string // absolute URL
	PubDate     time.Time
	Description string
	Content     string // HTML body, CDATA-wrapped on output
}

// WriteRSS marshals the feed as RSS 2.0 with content:encoded for HTML
// bodies and an atom:link self-reference.
func (f *Feed) WriteRSS(w io.Writer) error {
	doc := rssDoc{
		Version:    rssVersion,
		AtomNS:     nsAtom,
		ContentNS:  nsContent,
		Channel: rssChannel{
			Title:       f.Title,
			Description: f.Description,
			Link:        f.Link,
			Language:    f.Language,
		},
	}
	if !f.LastBuild.IsZero() {
		doc.Channel.LastBuild = f.LastBuild.UTC().Format(time.RFC1123Z)
	}
	if f.SelfLink != "" {
		doc.Channel.AtomLink = &atomLink{
			Href: f.SelfLink,
			Rel:  "self",
			Type: "application/rss+xml",
		}
	}
	for _, it := range f.Items {
		doc.Channel.Items = append(doc.Channel.Items, rssItem{
			Title:       it.Title,
			Link:        it.Link,
			GUID:        rssGUID{IsPermaLink: "true", Value: it.Link},
			PubDate:     it.PubDate.UTC().Format(time.RFC1123Z),
			Description: it.Description,
			Content:     cdataString{Value: it.Content},
		})
	}
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

// XML marshaling shapes.
type rssDoc struct {
	XMLName    xml.Name   `xml:"rss"`
	Version    string     `xml:"version,attr"`
	AtomNS     string     `xml:"xmlns:atom,attr"`
	ContentNS  string     `xml:"xmlns:content,attr"`
	Channel    rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Language    string    `xml:"language,omitempty"`
	LastBuild   string    `xml:"lastBuildDate,omitempty"`
	AtomLink    *atomLink `xml:"atom:link,omitempty"`
	Items       []rssItem `xml:"item"`
}

type atomLink struct {
	XMLName xml.Name `xml:"atom:link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr"`
	Type    string   `xml:"type,attr"`
}

type rssItem struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	GUID        rssGUID     `xml:"guid"`
	PubDate     string      `xml:"pubDate"`
	Description string      `xml:"description"`
	Content     cdataString `xml:"content:encoded"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type cdataString struct {
	Value string `xml:",cdata"`
}
