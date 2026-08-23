package rss

import (
	"strings"
	"testing"
)

func TestParseAtomCompleteSchema(t *testing.T) {
	atom := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <id>urn:uuid:feed-id</id>
  <title>Complete Feed</title>
  <updated>2025-01-08T12:00:00Z</updated>
  <subtitle>A subtitle</subtitle>
  <icon>https://example.com/icon.png</icon>
  <logo>https://example.com/logo.png</logo>
  <rights>© 2025</rights>
  <generator uri="https://gen.example" version="1.0">Gen</generator>
  <author><name>Jane</name><uri>https://jane.example</uri><email>j@e.com</email></author>
  <contributor><name>Contrib</name></contributor>
  <category term="tech" scheme="https://cats" label="Tech"/>
  <link href="https://example.com" rel="alternate"/>
  <entry>
    <id>urn:uuid:e1</id>
    <title>Entry 1</title>
    <updated>2025-01-08T12:00:00Z</updated>
    <content type="html">&lt;p&gt;hi&lt;/p&gt;</content>
    <summary type="text">sum</summary>
  </entry>
</feed>`

	feed, err := ParseAtom([]byte(atom))
	if err != nil {
		t.Fatalf("ParseAtom fail: %s", err)
	}

	if feed.ID != "urn:uuid:feed-id" {
		t.Errorf("feed.ID = %q", feed.ID)
	}
	if feed.Updated != "2025-01-08T12:00:00Z" {
		t.Errorf("feed.Updated = %q", feed.Updated)
	}
	if feed.Subtitle != "A subtitle" || feed.Icon == "" || feed.Logo == "" || feed.Rights != "© 2025" {
		t.Errorf("feed header fields = %+v", feed)
	}
	if feed.Generator == nil || feed.Generator.Name != "Gen" || feed.Generator.Version != "1.0" {
		t.Errorf("generator = %+v", feed.Generator)
	}
	if len(feed.Authors) != 1 || feed.Authors[0].Name != "Jane" {
		t.Errorf("authors = %+v", feed.Authors)
	}
	if len(feed.Contributors) != 1 {
		t.Errorf("contributors = %d", len(feed.Contributors))
	}
	if len(feed.Categories) != 1 || feed.Categories[0].Term != "tech" || feed.Categories[0].Scheme == "" {
		t.Errorf("categories = %+v", feed.Categories)
	}

	e := feed.Entries[0]
	if e.Content.Type != "html" || e.Content.Body != "<p>hi</p>" {
		t.Errorf("content = %+v", e.Content)
	}
	if e.Summary.Type != "text" || e.Summary.Body != "sum" {
		t.Errorf("summary = %+v", e.Summary)
	}
}

func TestParseRSSCompleteChannel(t *testing.T) {
	doc := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>F</title>
    <link>https://e.com</link>
    <description>d</description>
    <language>zh-cn</language>
    <copyright>© me</copyright>
    <managingEditor>ed@e.com</managingEditor>
    <webMaster>web@e.com</webMaster>
    <pubDate>Mon, 06 Jan 2025 00:00:00 GMT</pubDate>
    <lastBuildDate>Mon, 06 Jan 2025 01:00:00 GMT</lastBuildDate>
    <generator>Gen</generator>
    <docs>https://cyber.harvard.edu/rss/rss.html</docs>
    <ttl>60</ttl>
    <category domain="https://cat">news</category>
    <skipHours><hour>0</hour><hour>6</hour></skipHours>
    <skipDays><day>Monday</day></skipDays>
    <item>
      <title>I</title>
      <link>https://e.com/1</link>
      <guid isPermaLink="false">tag:e.com,2025:1</guid>
      <comments>https://e.com/1#comments</comments>
      <source url="https://src.example">Src</source>
    </item>
  </channel>
</rss>`

	feed, err := ParseRSS([]byte(doc))
	if err != nil {
		t.Fatalf("ParseRSS fail: %s", err)
	}
	ch := feed.Channel

	if ch.Language != "zh-cn" || ch.Copyright == "" || ch.ManagingEditor != "ed@e.com" {
		t.Errorf("channel header = %+v", ch)
	}
	if ch.TTL != 60 || ch.Generator != "Gen" || ch.LastBuildDate == "" {
		t.Errorf("ttl/generator/lastBuildDate: ttl=%d gen=%q lbd=%q", ch.TTL, ch.Generator, ch.LastBuildDate)
	}
	if len(ch.Category) != 1 || ch.Category[0].Domain != "https://cat" {
		t.Errorf("categories = %+v", ch.Category)
	}
	if len(ch.SkipHours) != 2 || ch.SkipHours[0] != 0 || ch.SkipHours[1] != 6 {
		t.Errorf("skipHours = %v", ch.SkipHours)
	}
	if len(ch.SkipDays) != 1 || ch.SkipDays[0] != "Monday" {
		t.Errorf("skipDays = %v", ch.SkipDays)
	}

	it := ch.Items[0]
	if it.GUID == nil || it.GUID.Value != "tag:e.com,2025:1" || it.GUID.IsPermaLink {
		t.Errorf("guid = %+v, want isPermaLink=false", it.GUID)
	}
	if it.Comments == "" || it.Source == nil || it.Source.URL != "https://src.example" {
		t.Errorf("comments/source: %q %+v", it.Comments, it.Source)
	}
}

func TestParseOPMLCompleteHead(t *testing.T) {
	doc := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Subs</title>
    <dateCreated>Mon, 06 Jan 2025 00:00:00 GMT</dateCreated>
    <dateModified>Mon, 06 Jan 2025 01:00:00 GMT</dateModified>
    <ownerName>Jane</ownerName>
    <ownerEmail>j@e.com</ownerEmail>
    <ownerId>http://jane.example</ownerId>
    <docs>http://opml.org/spec2.opml</docs>
    <expansionState>1,3,5</expansionState>
    <vertScrollState>1</vertScrollState>
    <windowTop>100</windowTop>
    <windowLeft>50</windowLeft>
    <windowBottom>500</windowBottom>
    <windowRight>600</windowRight>
  </head>
  <body>
    <outline type="rss" text="News" xmlUrl="https://n.example/rss" htmlUrl="https://n.example" description="news" category="/news" created="Mon, 06 Jan 2025" isComment="false"/>
  </body>
</opml>`

	opml, err := ParseOPML([]byte(doc))
	if err != nil {
		t.Fatalf("ParseOPML fail: %s", err)
	}
	h := opml.Head
	if h.Title != "Subs" || h.OwnerName != "Jane" || h.OwnerEmail != "j@e.com" || h.OwnerID == "" {
		t.Errorf("head owner fields = %+v", h)
	}
	if h.Docs == "" || h.VertScrollState != 1 || h.WindowTop != 100 || h.WindowBottom != 500 {
		t.Errorf("head misc = %+v", h)
	}

	o := opml.Body.Outlines[0]
	if o.Description != "news" || o.Category != "/news" || o.Created == "" {
		t.Errorf("outline attrs = %+v", o)
	}
	if o.IsComment == nil || *o.IsComment {
		t.Errorf("isComment = %v, want false", o.IsComment)
	}
	_ = strings.TrimSpace
}

func TestToAtomProducesRequiredFields(t *testing.T) {
	jf := &JSONFeed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       "T",
		FeedURL:     "https://e.com/feed.json",
		HomePageURL: "https://e.com",
		Items: []JSONFeedItem{
			{ID: "1", URL: "https://e.com/1", ContentHTML: "<p>c</p>", DatePublished: "2025-01-08T12:00:00Z"},
		},
	}
	feed := jf.ToAtom()

	if feed.ID == "" {
		t.Error("feed.ID (required) is empty")
	}
	if feed.Updated == "" {
		t.Error("feed.Updated (required) is empty")
	}
	e := feed.Entries[0]
	if e.Updated == "" {
		t.Error("entry.Updated (required) fell back empty")
	}
	if e.Content.Type != "html" {
		t.Errorf("content type = %q, want html", e.Content.Type)
	}
}
