// Package rss provides types and parsing functions for RSS, Atom, and OPML formats.
package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// RSS represents an RSS feed document.
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

// Channel represents the channel element of an RSS feed.
type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Items       []Item `xml:"item"`
}

// Item represents a single item in an RSS feed.
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"` //nolint:staticcheck // SA5008: valid RSS content namespace
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author"`
}

// Feed represents an Atom feed document.
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	Entries []Entry  `xml:"entry"`
}

// Entry represents a single entry in an Atom feed.
type Entry struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Updated string `xml:"updated"`
	Summary string `xml:"summary"`
	Content string `xml:"content"`
	Author  Author `xml:"author"`
	Links   []Link `xml:"link"`
}

// AlternateLink returns the href of the first link with rel="alternate",
// or the first link if none has that rel.
func (e *Entry) AlternateLink() string {
	for _, l := range e.Links {
		if l.Rel == "alternate" {
			return l.Href
		}
	}
	if len(e.Links) > 0 {
		return e.Links[0].Href
	}
	return ""
}

// Author represents an author element in an Atom feed.
type Author struct {
	Name  string `xml:"name"`
	Email string `xml:"email,omitempty"`
}

// Link represents a link element in an Atom feed.
type Link struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

// ParseRSS parses XML data into an RSS feed.
func ParseRSS(data []byte) (*RSS, error) {
	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("parse RSS: %w", err)
	}
	return &rss, nil
}

// ParseAtom parses XML data into an Atom feed.
func ParseAtom(data []byte) (*Feed, error) {
	var feed Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parse Atom: %w", err)
	}
	return &feed, nil
}

// FeedType represents the detected feed format.
type FeedType int

const (
	FeedTypeUnknown FeedType = iota
	FeedTypeRSS
	FeedTypeAtom
)

// DetectFeedType detects whether the data is RSS or Atom format.
func DetectFeedType(data []byte) FeedType {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return FeedTypeUnknown
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "rss", "RDF": // RSS 2.0 and RSS 1.0 (RDF)
			return FeedTypeRSS
		case "feed":
			return FeedTypeAtom
		}
	}
}

// Parse auto-detects the feed format and parses accordingly.
// Returns either *RSS or *Feed depending on the detected format.
func Parse(data []byte) (any, error) {
	switch DetectFeedType(data) {
	case FeedTypeRSS:
		return ParseRSS(data)
	case FeedTypeAtom:
		return ParseAtom(data)
	default:
		return nil, fmt.Errorf("unknown feed format: %s", strings.TrimSpace(string(data[:min(len(data), 64)])))
	}
}
