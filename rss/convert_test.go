package rss

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
)

// --- Date conversion helpers ---

func TestRFC822ToRFC3339(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Mon, 06 Jan 2025 00:00:00 +0000", "2025-01-06T00:00:00Z"},
		{"Tue, 07 Jan 2025 00:00:00 +0000", "2025-01-07T00:00:00Z"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		got := rfc822ToRFC3339(tt.input)
		if got != tt.want {
			t.Errorf("rfc822ToRFC3339(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRFC3339ToRFC822(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2025-01-06T00:00:00Z", "Mon, 06 Jan 2025 00:00:00 +0000"},
		{"2025-01-06T12:00:00+08:00", "Mon, 06 Jan 2025 12:00:00 +0800"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		got := rfc3339ToRFC822(tt.input)
		if got != tt.want {
			t.Errorf("rfc3339ToRFC822(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- RSS → JSONFeed ---

func TestRSS_ToJSONFeed(t *testing.T) {
	rss, err := ParseRSS([]byte(rssXML))
	if err != nil {
		t.Fatalf("ParseRSS() error = %v", err)
	}

	jf := rss.ToJSONFeed()
	assertEqual(t, "version", jf.Version, "https://jsonfeed.org/version/1.1")
	assertEqual(t, "title", jf.Title, "Test Feed")
	assertEqual(t, "description", jf.Description, "A test RSS feed")
	assertEqual(t, "home_page_url", jf.HomePageURL, "https://example.com")
	assertEqual(t, "items count", len(jf.Items), 2)

	// Item 1: has content:encoded, author, guid, pubDate
	item1 := jf.Items[0]
	assertEqual(t, "item[0].ID", item1.ID, "https://example.com/1")
	assertEqual(t, "item[0].URL", item1.URL, "https://example.com/1")
	assertEqual(t, "item[0].Title", item1.Title, "First Post")
	assertEqual(t, "item[0].ContentHTML", item1.ContentHTML, "<p>Full content</p>") // content:encoded
	assertEqual(t, "item[0].DatePublished", item1.DatePublished, "2025-01-06T00:00:00Z")
	assertEqual(t, "item[0].authors count", len(item1.Authors), 1)
	assertEqual(t, "item[0].author name", item1.Authors[0].Name, "john@example.com")

	// Item 2: no author, no guid, no content:encoded → uses description
	item2 := jf.Items[1]
	assertEqual(t, "item[1].ID", item2.ID, "")
	assertEqual(t, "item[1].Title", item2.Title, "Second Post")
	assertEqual(t, "item[1].ContentHTML", item2.ContentHTML, "Another summary") // fallback to description
	assertEqual(t, "item[1].DatePublished", item2.DatePublished, "2025-01-07T00:00:00Z")
	assertEqual(t, "item[1].authors count", len(item2.Authors), 0)
}

// --- Atom → JSONFeed ---

func TestAtom_ToJSONFeed(t *testing.T) {
	feed, err := ParseAtom([]byte(atomXML))
	if err != nil {
		t.Fatalf("ParseAtom() error = %v", err)
	}

	jf := feed.ToJSONFeed()
	assertEqual(t, "version", jf.Version, "https://jsonfeed.org/version/1.1")
	assertEqual(t, "title", jf.Title, "Atom Test Feed")
	assertEqual(t, "items count", len(jf.Items), 2)

	// Entry 1
	item1 := jf.Items[0]
	assertEqual(t, "item[0].ID", item1.ID, "urn:uuid:1")
	assertEqual(t, "item[0].Title", item1.Title, "Atom Entry One")
	assertEqual(t, "item[0].Summary", item1.Summary, "A brief summary")
	assertEqual(t, "item[0].ContentHTML", item1.ContentHTML, "Full entry content here")
	assertEqual(t, "item[0].URL", item1.URL, "https://example.com/atom/1") // first alternate link
	assertEqual(t, "item[0].authors count", len(item1.Authors), 1)
	assertEqual(t, "item[0].author name", item1.Authors[0].Name, "Jane")

	// Entry 2: minimal
	item2 := jf.Items[1]
	assertEqual(t, "item[1].ID", item2.ID, "urn:uuid:2")
	assertEqual(t, "item[1].Title", item2.Title, "Atom Entry Two")
	assertEqual(t, "item[1].URL", item2.URL, "https://example.com/atom/2")
}

// --- JSONFeed → RSS ---

func TestJSONFeed_ToRSS(t *testing.T) {
	jf, err := ParseJSONFeed([]byte(jsonFeedV11))
	if err != nil {
		t.Fatalf("ParseJSONFeed() error = %v", err)
	}

	rss := jf.ToRSS()
	assertEqual(t, "channel title", rss.Channel.Title, "My Test Feed")
	assertEqual(t, "channel description", rss.Channel.Description, "A test JSON Feed")
	assertEqual(t, "channel link", rss.Channel.Link, "https://example.org/")
	assertEqual(t, "items count", len(rss.Channel.Items), 2)

	// Item 1: has content_html, authors, attachments, tags
	item1 := rss.Channel.Items[0]
	assertEqual(t, "item[0].Title", item1.Title, "First Post")
	assertEqual(t, "item[0].Link", item1.Link, "https://example.org/post/1")
	assertEqual(t, "item[0].GUID", item1.GUID, "1")
	assertEqual(t, "item[0].Content", item1.Content, "<p>Hello, world!</p>") // content_html → content:encoded
	assertEqual(t, "item[0].Author", item1.Author, "Jane Doe")
	assertEqual(t, "item[0].PubDate", item1.PubDate, "Mon, 06 Jan 2025 00:00:00 +0000")
	assertEqual(t, "item[0].enclosure url", item1.Enclosure.URL, "https://example.org/audio/1.mp3")
	assertEqual(t, "item[0].enclosure type", item1.Enclosure.Type, "audio/mpeg")
	assertEqual(t, "item[0].enclosure size", item1.Enclosure.Length, int64(12345678))
	assertEqual(t, "item[0].categories count", len(item1.Categories), 2)
	assertEqual(t, "item[0].category[0]", item1.Categories[0].Value, "tech")
	assertEqual(t, "item[0].category[1]", item1.Categories[1].Value, "go")

	// Item 2: has content_text only
	item2 := rss.Channel.Items[1]
	assertEqual(t, "item[1].Title", item2.Title, "")
	assertEqual(t, "item[1].GUID", item2.GUID, "2")
	assertEqual(t, "item[1].Description", item2.Description, "Plain text content.") // content_text → description
}

// --- JSONFeed → Atom ---

func TestJSONFeed_ToAtom(t *testing.T) {
	jf, err := ParseJSONFeed([]byte(jsonFeedV11))
	if err != nil {
		t.Fatalf("ParseJSONFeed() error = %v", err)
	}

	feed := jf.ToAtom()
	assertEqual(t, "title", feed.Title, "My Test Feed")

	// Feed-level links
	assertEqual(t, "feed links count", len(feed.Links), 3) // alternate + self + hub
	assertEqual(t, "feed link[0] rel", feed.Links[0].Rel, "alternate")
	assertEqual(t, "feed link[0] href", feed.Links[0].Href, "https://example.org/")
	assertEqual(t, "feed link[1] rel", feed.Links[1].Rel, "self")
	assertEqual(t, "feed link[1] href", feed.Links[1].Href, "https://example.org/feed.json")
	assertEqual(t, "feed link[2] rel", feed.Links[2].Rel, "hub")
	assertEqual(t, "feed link[2] href", feed.Links[2].Href, "https://example.org/hub")

	assertEqual(t, "entries count", len(feed.Entries), 2)

	// Entry 1
	e1 := feed.Entries[0]
	assertEqual(t, "entry[0].Title", e1.Title, "First Post")
	assertEqual(t, "entry[0].ID", e1.ID, "1")
	assertEqual(t, "entry[0].Content", e1.Content, "<p>Hello, world!</p>")
	assertEqual(t, "entry[0].Summary", e1.Summary, "A short summary")
	assertEqual(t, "entry[0].Published", e1.Published, "2025-01-06T00:00:00Z")
	assertEqual(t, "entry[0].Updated", e1.Updated, "2025-01-06T12:00:00Z")
	assertEqual(t, "entry[0].Author.Name", e1.Author.Name, "Jane Doe")

	// Entry 1 links: alternate + enclosure
	var alternateCount, enclosureCount int
	for _, l := range e1.Links {
		switch l.Rel {
		case "alternate":
			alternateCount++
			assertEqual(t, "entry[0] alternate href", l.Href, "https://example.org/post/1")
		case "enclosure":
			enclosureCount++
			assertEqual(t, "entry[0] enclosure href", l.Href, "https://example.org/audio/1.mp3")
			assertEqual(t, "entry[0] enclosure type", l.Type, "audio/mpeg")
		}
	}
	assertEqual(t, "entry[0] alternate links", alternateCount, 1)
	assertEqual(t, "entry[0] enclosure links", enclosureCount, 1)

	// Entry 1 categories
	assertEqual(t, "entry[0].categories count", len(e1.Categories), 2)
	assertEqual(t, "entry[0].category[0]", e1.Categories[0].Term, "tech")
	assertEqual(t, "entry[0].category[1]", e1.Categories[1].Term, "go")

	// Entry 2: minimal, content_text → content
	e2 := feed.Entries[1]
	assertEqual(t, "entry[1].ID", e2.ID, "2")
	assertEqual(t, "entry[1].Content", e2.Content, "Plain text content.") // content_text fallback
	assertEqual(t, "entry[1].Published", e2.Published, "2025-01-07T00:00:00Z")
}

// --- Round-trip tests ---

func TestRoundTrip_RSS(t *testing.T) {
	rss, err := ParseRSS([]byte(rssXML))
	if err != nil {
		t.Fatalf("ParseRSS() error = %v", err)
	}

	// RSS → JSONFeed → RSS
	jf := rss.ToJSONFeed()
	rss2 := jf.ToRSS()

	assertEqual(t, "channel title", rss2.Channel.Title, rss.Channel.Title)
	assertEqual(t, "channel link", rss2.Channel.Link, rss.Channel.Link)
	assertEqual(t, "items count", len(rss2.Channel.Items), len(rss.Channel.Items))

	for i, orig := range rss.Channel.Items {
		got := rss2.Channel.Items[i]
		assertEqual(t, "item title", got.Title, orig.Title)
		assertEqual(t, "item link", got.Link, orig.Link)
		assertEqual(t, "item guid", got.GUID, orig.GUID)
		assertEqual(t, "item author", got.Author, orig.Author)
	}
}

func TestRoundTrip_Atom(t *testing.T) {
	feed, err := ParseAtom([]byte(atomXML))
	if err != nil {
		t.Fatalf("ParseAtom() error = %v", err)
	}

	// Atom → JSONFeed → Atom
	jf := feed.ToJSONFeed()
	feed2 := jf.ToAtom()

	assertEqual(t, "title", feed2.Title, feed.Title)
	assertEqual(t, "entries count", len(feed2.Entries), len(feed.Entries))

	for i, orig := range feed.Entries {
		got := feed2.Entries[i]
		assertEqual(t, "entry title", got.Title, orig.Title)
		assertEqual(t, "entry id", got.ID, orig.ID)
		assertEqual(t, "entry content", got.Content, orig.Content)
		assertEqual(t, "entry summary", got.Summary, orig.Summary)
	}
}

// --- Conversion with enclosure/category ---

const rssWithEnclosureAndCategory = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Podcast</title>
    <link>https://podcast.example.com</link>
    <item>
      <title>Episode 1</title>
      <link>https://podcast.example.com/ep1</link>
      <guid>https://podcast.example.com/ep1</guid>
      <description>First episode</description>
      <pubDate>Mon, 06 Jan 2025 00:00:00 +0000</pubDate>
      <enclosure url="https://podcast.example.com/ep1.mp3" length="12345678" type="audio/mpeg"/>
      <category>technology</category>
      <category domain="topic">golang</category>
    </item>
  </channel>
</rss>`

func TestRSS_ToJSONFeed_EnclosureAndCategory(t *testing.T) {
	rss, err := ParseRSS([]byte(rssWithEnclosureAndCategory))
	if err != nil {
		t.Fatalf("ParseRSS() error = %v", err)
	}

	jf := rss.ToJSONFeed()
	assertEqual(t, "items count", len(jf.Items), 1)

	item := jf.Items[0]
	assertEqual(t, "title", item.Title, "Episode 1")
	assertEqual(t, "url", item.URL, "https://podcast.example.com/ep1")

	// Enclosure → attachment
	assertEqual(t, "attachments count", len(item.Attachments), 1)
	assertEqual(t, "attachment url", item.Attachments[0].URL, "https://podcast.example.com/ep1.mp3")
	assertEqual(t, "attachment mime_type", item.Attachments[0].MimeType, "audio/mpeg")
	assertEqual(t, "attachment size", item.Attachments[0].SizeInBytes, int64(12345678))

	// Categories → tags
	assertEqual(t, "tags count", len(item.Tags), 2)
	assertEqual(t, "tag[0]", item.Tags[0], "technology")
	assertEqual(t, "tag[1]", item.Tags[1], "golang")
}

const atomWithPublishedAndCategory = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Blog</title>
  <link href="https://blog.example.com/" rel="alternate"/>
  <link href="https://blog.example.com/feed.atom" rel="self"/>
  <link href="https://blog.example.com/hub" rel="hub"/>
  <entry>
    <title>Post</title>
    <id>urn:uuid:42</id>
    <published>2025-01-05T10:00:00Z</published>
    <updated>2025-01-05T12:00:00Z</updated>
    <summary>Brief</summary>
    <content>Full content</content>
    <author><name>Alice</name><uri>https://alice.example.com</uri></author>
    <link href="https://blog.example.com/post/42" rel="alternate"/>
    <link href="https://cdn.example.com/audio.mp3" rel="enclosure" type="audio/mpeg" length="999"/>
    <category term="go"/>
    <category term="testing"/>
  </entry>
</feed>`

func TestAtom_ToJSONFeed_Full(t *testing.T) {
	feed, err := ParseAtom([]byte(atomWithPublishedAndCategory))
	if err != nil {
		t.Fatalf("ParseAtom() error = %v", err)
	}

	jf := feed.ToJSONFeed()
	assertEqual(t, "title", jf.Title, "Blog")
	assertEqual(t, "home_page_url", jf.HomePageURL, "https://blog.example.com/")
	assertEqual(t, "feed_url", jf.FeedURL, "https://blog.example.com/feed.atom")
	assertEqual(t, "hubs count", len(jf.Hubs), 1)
	assertEqual(t, "hub url", jf.Hubs[0].URL, "https://blog.example.com/hub")

	assertEqual(t, "items count", len(jf.Items), 1)
	item := jf.Items[0]
	assertEqual(t, "id", item.ID, "urn:uuid:42")
	assertEqual(t, "title", item.Title, "Post")
	assertEqual(t, "url", item.URL, "https://blog.example.com/post/42")
	assertEqual(t, "summary", item.Summary, "Brief")
	assertEqual(t, "content_html", item.ContentHTML, "Full content")
	assertEqual(t, "date_published", item.DatePublished, "2025-01-05T10:00:00Z")
	assertEqual(t, "date_modified", item.DateModified, "2025-01-05T12:00:00Z")
	assertEqual(t, "author name", item.Authors[0].Name, "Alice")
	assertEqual(t, "author url", item.Authors[0].URL, "https://alice.example.com")
	assertEqual(t, "attachments count", len(item.Attachments), 1)
	assertEqual(t, "attachment url", item.Attachments[0].URL, "https://cdn.example.com/audio.mp3")
	assertEqual(t, "attachment mime", item.Attachments[0].MimeType, "audio/mpeg")
	assertEqual(t, "tags count", len(item.Tags), 2)
	assertEqual(t, "tag[0]", item.Tags[0], "go")
	assertEqual(t, "tag[1]", item.Tags[1], "testing")
}

func TestJSONFeed_ToAtom_ExternalURL(t *testing.T) {
	data := `{"version":"https://jsonfeed.org/version/1.1","title":"Links","items":[{"id":"1","url":"https://a.com/1","external_url":"https://b.com/original","title":"Linkblog"}]}`
	jf, err := ParseJSONFeed([]byte(data))
	if err != nil {
		t.Fatalf("ParseJSONFeed() error = %v", err)
	}

	feed := jf.ToAtom()
	e := feed.Entries[0]
	assertEqual(t, "entry links count", len(e.Links), 2)

	var alternate, related string
	for _, l := range e.Links {
		switch l.Rel {
		case "alternate":
			alternate = l.Href
		case "related":
			related = l.Href
		}
	}
	assertEqual(t, "alternate href", alternate, "https://a.com/1")
	assertEqual(t, "related href", related, "https://b.com/original")
}

// --- Serialize converted output ---

func TestRSS_ToJSONFeed_Serialize(t *testing.T) {
	rss, _ := ParseRSS([]byte(rssXML))
	jf := rss.ToJSONFeed()

	data, err := json.Marshal(jf)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if !strings.Contains(string(data), `"version"`) {
		t.Error("serialized JSON should contain version field")
	}
	if !strings.Contains(string(data), `"items"`) {
		t.Error("serialized JSON should contain items field")
	}
}

func TestJSONFeed_ToRSS_Serialize(t *testing.T) {
	jf, _ := ParseJSONFeed([]byte(jsonFeedV11))
	rss := jf.ToRSS()

	data, err := xml.Marshal(rss)
	if err != nil {
		t.Fatalf("xml.Marshal() error = %v", err)
	}
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "<rss>") {
		t.Error("serialized XML should contain <rss>")
	}
	if !strings.Contains(xmlStr, "<channel>") {
		t.Error("serialized XML should contain <channel>")
	}
}

func TestJSONFeed_ToAtom_Serialize(t *testing.T) {
	jf, _ := ParseJSONFeed([]byte(jsonFeedV11))
	feed := jf.ToAtom()

	data, err := xml.Marshal(feed)
	if err != nil {
		t.Fatalf("xml.Marshal() error = %v", err)
	}
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "<feed>") {
		t.Error("serialized XML should contain <feed>")
	}
	if !strings.Contains(xmlStr, "<entry>") {
		t.Error("serialized XML should contain <entry>")
	}
}
