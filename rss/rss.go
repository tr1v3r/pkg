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

// Channel represents the channel element of an RSS feed (RSS 2.0 §5).
type Channel struct {
	// Required.
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`

	// Optional but common.
	Language       string     `xml:"language,omitempty"`
	Copyright      string     `xml:"copyright,omitempty"`
	ManagingEditor string     `xml:"managingEditor,omitempty"`
	WebMaster      string     `xml:"webMaster,omitempty"`
	PubDate        string     `xml:"pubDate,omitempty"`
	LastBuildDate  string     `xml:"lastBuildDate,omitempty"`
	Category       []Category `xml:"category,omitempty"`
	Generator      string     `xml:"generator,omitempty"`
	Docs           string     `xml:"docs,omitempty"`
	Cloud          *Cloud     `xml:"cloud,omitempty"`
	TTL            int        `xml:"ttl,omitempty"`
	Image          *Image     `xml:"image,omitempty"`
	Rating         string     `xml:"rating,omitempty"`
	TextInput      *TextInput `xml:"textInput,omitempty"`
	SkipHours      []int      `xml:"skipHours>hour"`
	SkipDays       []string   `xml:"skipDays>day"`

	Items []Item `xml:"item"`
}

// Cloud represents the channel's cloud element (RSS 2.0 §5.6).
type Cloud struct {
	Domain            string `xml:"domain,attr"`
	Port              int    `xml:"port,attr"`
	Path              string `xml:"path,attr"`
	RegisterProcedure string `xml:"registerProcedure,attr"`
	Protocol          string `xml:"protocol,attr"`
}

// TextInput represents the channel's textInput element (RSS 2.0 §5.7).
type TextInput struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Name        string `xml:"name"`
	Link        string `xml:"link"`
}

// Image represents the image element of an RSS feed.
type Image struct {
	URL   string `xml:"url"`
	Title string `xml:"title,omitempty"`
	Link  string `xml:"link,omitempty"`
}

// GUID represents an item's guid element (RSS 2.0 §4.2.10). isPermaLink
// defaults to true per the spec; the attribute is emitted only when false.
type GUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr,omitempty"`
	Value       string `xml:",chardata"`
}

// Source represents an item's source element (RSS 2.0 §4.2.11).
type Source struct {
	URL   string `xml:"url,attr"`
	Value string `xml:",chardata"`
}

// Item represents a single item in an RSS feed (RSS 2.0 §4).
type Item struct {
	Title       string     `xml:"title,omitempty"`
	Link        string     `xml:"link,omitempty"`
	Description string     `xml:"description,omitempty"`
	Content     string     `xml:"http://purl.org/rss/1.0/modules/content/ encoded"` //nolint:staticcheck // SA5008
	Author      string     `xml:"author,omitempty"`
	Category    []Category `xml:"category,omitempty"`
	Comments    string     `xml:"comments,omitempty"`
	Enclosure   *Enclosure `xml:"enclosure,omitempty"`
	GUID        *GUID      `xml:"guid,omitempty"`
	PubDate     string     `xml:"pubDate,omitempty"`
	Source      *Source    `xml:"source,omitempty"`
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

// DeduplicateItems removes duplicate items from the channel.
// It uses GUID as the primary key; if GUID is empty, it falls back to Link.
// The first occurrence of each item is kept.
func (ch *Channel) DeduplicateItems() {
	seen := make(map[string]struct{})
	// fresh backing array: reusing ch.Items[:0] would overwrite any slice
	// aliases the caller may still hold from before the call
	filtered := make([]Item, 0, len(ch.Items))
	for _, item := range ch.Items {
		key := ""
		if item.GUID != nil {
			key = item.GUID.Value
		}
		if key == "" {
			key = item.Link
		}
		if key == "" {
			filtered = append(filtered, item)
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, item)
	}
	ch.Items = filtered
}

// Feed represents an Atom feed document (RFC 4287).
type Feed struct {
	XMLName xml.Name `xml:"feed"`

	// Required by RFC 4287 §4.1.1: id, title, updated.
	ID      string `xml:"id"`
	Title   string `xml:"title"`
	Updated string `xml:"updated"`

	// Required unless every entry carries an author.
	Authors []Author `xml:"author"`

	Links []Link `xml:"link"`

	// Recommended.
	Subtitle  string `xml:"subtitle,omitempty"` // aka tagline
	Icon      string `xml:"icon,omitempty"`     // square, ~human-scale
	Logo      string `xml:"logo,omitempty"`     // rectangle, ~2:1
	Rights    string `xml:"rights,omitempty"`   // aka copyright
	Generator *struct {
		Name    string `xml:",chardata"`
		URI     string `xml:"uri,attr,omitempty"`
		Version string `xml:"version,attr,omitempty"`
	} `xml:"generator,omitempty"`

	// Optional.
	Categories   []AtomCategory `xml:"category,omitempty"`
	Contributors []Author       `xml:"contributor,omitempty"`
	Language     string         `xml:"lang,attr,omitempty"` // xml:lang
	Base         string         `xml:"base,attr,omitempty"` // xml:base
	Entries      []Entry        `xml:"entry"`
}

// Entry represents a single entry in an Atom feed (RFC 4287 §4.1.2).
type Entry struct {
	// Required: id, title, updated.
	ID      string `xml:"id"`
	Title   string `xml:"title"`
	Updated string `xml:"updated"`

	Authors []Author `xml:"author"`

	Published string `xml:"published,omitempty"`
	Rights    string `xml:"rights,omitempty"`
	Source    *struct {
		ID    string `xml:"id,omitempty"`
		Title string `xml:"title,omitempty"`
		Links []Link `xml:"link"`
	} `xml:"source,omitempty"`

	Summary AtomText `xml:"summary,omitempty"`
	Content AtomText `xml:"content,omitempty"`

	Links        []Link         `xml:"link"`
	Categories   []AtomCategory `xml:"category,omitempty"`
	Contributors []Author       `xml:"contributor,omitempty"`
}

// AtomCategory represents an Atom category element (RFC 4287 §3.4.2.2).
type AtomCategory struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr,omitempty"`
	Label  string `xml:"label,attr,omitempty"`
}

// AtomText models Text constructs (RFC 4287 §3.1): text, html or xhtml,
// plus the optional src/type attributes of content (§4.1.3).
type AtomText struct {
	Type string `xml:"type,attr,omitempty"` // "text" (default), "html" or "xhtml"; for content also mime types
	Src  string `xml:"src,attr,omitempty"`  // out-of-line content URI
	Body string `xml:",chardata"`
}

// String returns the text body, satisfying fmt.Stringer.
func (t AtomText) String() string { return t.Body }

// Link relation names used across RSS/Atom/JSON Feed conversions.
const (
	relAlternate = "alternate"
	relRelated   = "related"
	relEnclosure = "enclosure"
)

// AlternateLink returns the href of the first link with rel="alternate",
// or the first link if none has that rel.
func (e *Entry) AlternateLink() string {
	for _, l := range e.Links {
		if l.Rel == relAlternate {
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
	// FeedTypeUnknown identifies a feed format.
	FeedTypeUnknown FeedType = iota
	// FeedTypeRSS identifies a feed format.
	FeedTypeRSS
	// FeedTypeAtom identifies a feed format.
	FeedTypeAtom
	// FeedTypeJSON identifies a feed format.
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

// ParseTyped auto-detects the feed format and parses into the concrete type
// named by T (*RSS, *Feed or *JSONFeed). It errors at runtime when the data
// is not of that format, giving callers a compile-time-typed alternative to
// the any-returning Parse.
func ParseTyped[T any](data []byte) (T, error) {
	var zero T

	want := FeedTypeUnknown
	switch any(zero).(type) {
	case *RSS:
		want = FeedTypeRSS
	case *Feed:
		want = FeedTypeAtom
	case *JSONFeed:
		want = FeedTypeJSON
	}

	got := DetectFeedType(data)
	if want == FeedTypeUnknown {
		return zero, fmt.Errorf("ParseTyped: type parameter must be *rss.RSS, *rss.Feed or *rss.JSONFeed")
	}
	if got != want {
		return zero, fmt.Errorf("ParseTyped: feed is %s, not %s", got, want)
	}

	parsed, err := Parse(data)
	if err != nil {
		return zero, err
	}
	return parsed.(T), nil
}

// Feed type names for error messages and String().
const (
	feedNameRSS    = "RSS"
	feedNameAtom   = "Atom"
	feedNameJSON   = "JSON Feed"
	feedNameUnknow = "unknown"
)

// String makes feed types readable in error messages.
func (t FeedType) String() string {
	switch t {
	case FeedTypeRSS:
		return feedNameRSS
	case FeedTypeAtom:
		return feedNameAtom
	case FeedTypeJSON:
		return feedNameJSON
	default:
		return feedNameUnknow
	}
}
