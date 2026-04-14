package rss

import (
	"testing"
)

// --- RSS parsing ---

const rssXML = `
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <description>A test RSS feed</description>
    <link>https://example.com</link>
    <item>
      <title>First Post</title>
      <link>https://example.com/1</link>
      <description>Short summary</description>
      <content:encoded xmlns:content="http://purl.org/rss/1.0/modules/content/">&lt;p&gt;Full content&lt;/p&gt;</content:encoded>
      <pubDate>Mon, 06 Jan 2025 00:00:00 +0000</pubDate>
      <guid>https://example.com/1</guid>
      <author>john@example.com</author>
    </item>
    <item>
      <title>Second Post</title>
      <link>https://example.com/2</link>
      <description>Another summary</description>
      <pubDate>Tue, 07 Jan 2025 00:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

func TestParseRSS(t *testing.T) {
	feed, err := ParseRSS([]byte(rssXML))
	if err != nil {
		t.Fatalf("ParseRSS() error = %v", err)
	}

	assertEqual(t, "channel title", feed.Channel.Title, "Test Feed")
	assertEqual(t, "channel link", feed.Channel.Link, "https://example.com")
	assertEqual(t, "item count", len(feed.Channel.Items), 2)

	item1 := feed.Channel.Items[0]
	assertEqual(t, "item[0].Title", item1.Title, "First Post")
	assertEqual(t, "item[0].Link", item1.Link, "https://example.com/1")
	assertEqual(t, "item[0].GUID", item1.GUID, "https://example.com/1")
	assertEqual(t, "item[0].Author", item1.Author, "john@example.com")

	item2 := feed.Channel.Items[1]
	assertEqual(t, "item[1].Title", item2.Title, "Second Post")
	assertEqual(t, "item[1].GUID", item2.GUID, "")
	assertEqual(t, "item[1].Author", item2.Author, "")
}

func TestParseRSS_InvalidXML(t *testing.T) {
	_, err := ParseRSS([]byte("not xml"))
	if err == nil {
		t.Fatal("ParseRSS() expected error for invalid XML")
	}
}

// --- Atom parsing ---

const atomXML = `
<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Test Feed</title>
  <entry>
    <title>Atom Entry One</title>
    <id>urn:uuid:1</id>
    <updated>2025-01-06T00:00:00Z</updated>
    <summary>A brief summary</summary>
    <content>Full entry content here</content>
    <author><name>Jane</name><email>jane@example.com</email></author>
    <link href="https://example.com/atom/1" rel="alternate" type="text/html"/>
    <link href="https://example.com/atom/1.json" rel="alternate" type="application/json"/>
  </entry>
  <entry>
    <title>Atom Entry Two</title>
    <id>urn:uuid:2</id>
    <updated>2025-01-07T00:00:00Z</updated>
    <link href="https://example.com/atom/2"/>
  </entry>
</feed>`

func TestParseAtom(t *testing.T) {
	feed, err := ParseAtom([]byte(atomXML))
	if err != nil {
		t.Fatalf("ParseAtom() error = %v", err)
	}

	assertEqual(t, "feed title", feed.Title, "Atom Test Feed")
	assertEqual(t, "entry count", len(feed.Entries), 2)

	e1 := feed.Entries[0]
	assertEqual(t, "entry[0].Title", e1.Title, "Atom Entry One")
	assertEqual(t, "entry[0].ID", e1.ID, "urn:uuid:1")
	assertEqual(t, "entry[0].Summary", e1.Summary, "A brief summary")
	assertEqual(t, "entry[0].Content", e1.Content, "Full entry content here")
	assertEqual(t, "entry[0].Author.Name", e1.Author.Name, "Jane")
	assertEqual(t, "entry[0].Author.Email", e1.Author.Email, "jane@example.com")
	assertEqual(t, "entry[0] links", len(e1.Links), 2)
	assertEqual(t, "entry[0].AlternateLink()", e1.AlternateLink(), "https://example.com/atom/1")

	e2 := feed.Entries[1]
	assertEqual(t, "entry[1].AlternateLink()", e2.AlternateLink(), "https://example.com/atom/2")
}

func TestParseAtom_InvalidXML(t *testing.T) {
	_, err := ParseAtom([]byte("<broken"))
	if err == nil {
		t.Fatal("ParseAtom() expected error for invalid XML")
	}
}

func TestAlternateLink_NoLinks(t *testing.T) {
	e := &Entry{}
	assertEqual(t, "AlternateLink with no links", e.AlternateLink(), "")
}

func TestAlternateLink_NoAlternate(t *testing.T) {
	e := &Entry{Links: []Link{{Href: "https://example.com/x", Rel: "enclosure"}}}
	assertEqual(t, "AlternateLink fallback", e.AlternateLink(), "https://example.com/x")
}

// --- DetectFeedType & Parse ---

func TestDetectFeedType(t *testing.T) {
	tests := []struct {
		name string
		data string
		want FeedType
	}{
		{"RSS 2.0", rssXML, FeedTypeRSS},
		{"Atom", atomXML, FeedTypeAtom},
		{"RSS 1.0 RDF", `<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"><channel/></rdf:RDF>`, FeedTypeRSS},
		{"JSON Feed 1.1", jsonFeedV11, FeedTypeJSON},
		{"JSON Feed 1.0", jsonFeedV10, FeedTypeJSON},
		{"JSON not a feed", `{"hello":"world"}`, FeedTypeUnknown},
		{"unknown", `<html><body/></html>`, FeedTypeUnknown},
		{"empty", ``, FeedTypeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFeedType([]byte(tt.data))
			if got != tt.want {
				t.Errorf("DetectFeedType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse_AutoDetect(t *testing.T) {
	result, err := Parse([]byte(rssXML))
	if err != nil {
		t.Fatalf("Parse(RSS) error = %v", err)
	}
	if _, ok := result.(*RSS); !ok {
		t.Fatalf("Parse(RSS) returned %T, want *RSS", result)
	}

	result, err = Parse([]byte(atomXML))
	if err != nil {
		t.Fatalf("Parse(Atom) error = %v", err)
	}
	if _, ok := result.(*Feed); !ok {
		t.Fatalf("Parse(Atom) returned %T, want *Feed", result)
	}

	result, err = Parse([]byte(jsonFeedV11))
	if err != nil {
		t.Fatalf("Parse(JSON) error = %v", err)
	}
	if _, ok := result.(*JSONFeed); !ok {
		t.Fatalf("Parse(JSON) returned %T, want *JSONFeed", result)
	}
}

func TestParse_UnknownFormat(t *testing.T) {
	_, err := Parse([]byte(`<html><body>hello</body></html>`))
	if err == nil {
		t.Fatal("Parse() expected error for unknown format")
	}
}
