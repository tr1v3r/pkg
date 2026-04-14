// Package rss provides types and parsing functions for RSS, Atom, JSON Feed, and OPML formats.
package rss

import (
	"bytes"
	"encoding/json"
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
	Enclosure   *Enclosure  `xml:"enclosure,omitempty"`
	Categories  []Category  `xml:"category,omitempty"`
}

// Enclosure represents an RSS enclosure element.
type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// Category represents an RSS category element.
type Category struct {
	Domain string `xml:"domain,attr,omitempty"`
	Value  string `xml:",chardata"`
}

// Feed represents an Atom feed document.
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	Links   []Link   `xml:"link"`
	Entries []Entry  `xml:"entry"`
}

// Entry represents a single entry in an Atom feed.
type Entry struct {
	Title       string         `xml:"title"`
	ID          string         `xml:"id"`
	Published   string         `xml:"published"`
	Updated     string         `xml:"updated"`
	Summary     string         `xml:"summary"`
	Content     string         `xml:"content"`
	Author      Author         `xml:"author"`
	Links       []Link         `xml:"link"`
	Categories  []AtomCategory `xml:"category,omitempty"`
}

// AtomCategory represents an Atom category element.
type AtomCategory struct {
	Term string `xml:"term,attr"`
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
	URI   string `xml:"uri,omitempty"`
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

// ParseJSONFeed parses JSON data into a JSON Feed document.
func ParseJSONFeed(data []byte) (*JSONFeed, error) {
	var feed JSONFeed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parse JSON Feed: %w", err)
	}
	return &feed, nil
}

// FeedType represents the detected feed format.
type FeedType int

const (
	FeedTypeUnknown FeedType = iota
	FeedTypeRSS
	FeedTypeAtom
	FeedTypeJSON
)

// DetectFeedType detects whether the data is RSS, Atom, or JSON Feed format.
func DetectFeedType(data []byte) FeedType {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return FeedTypeUnknown
	}

	// JSON Feed documents are JSON objects with a "version" field starting with "https://jsonfeed.org/version/".
	if trimmed[0] == '{' {
		var obj map[string]any
		if err := json.Unmarshal(trimmed, &obj); err == nil {
			if v, ok := obj["version"].(string); ok {
				if strings.HasPrefix(v, "https://jsonfeed.org/version/") {
					return FeedTypeJSON
				}
			}
		}
		return FeedTypeUnknown
	}

	// XML detection: RSS 2.0, RSS 1.0 (RDF), and Atom.
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
		case "rss", "RDF":
			return FeedTypeRSS
		case "feed":
			return FeedTypeAtom
		}
	}
}

// Parse auto-detects the feed format and parses accordingly.
// Returns *RSS, *Feed (Atom), or *JSONFeed depending on the detected format.
func Parse(data []byte) (any, error) {
	switch DetectFeedType(data) {
	case FeedTypeRSS:
		return ParseRSS(data)
	case FeedTypeAtom:
		return ParseAtom(data)
	case FeedTypeJSON:
		return ParseJSONFeed(data)
	default:
		return nil, fmt.Errorf("unknown feed format: %s", strings.TrimSpace(string(data[:min(len(data), 64)])))
	}
}
