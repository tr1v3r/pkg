package rss

import (
	"testing"
)

// --- OPML parsing ---

const opmlXML = `
<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>My Feeds</title>
    <dateCreated>2025-01-01</dateCreated>
  </head>
  <body>
    <outline text="Tech" title="Tech">
      <outline type="rss" text="Go Blog" xmlUrl="https://go.dev/blog/feed.atom" htmlUrl="https://go.dev/blog"/>
      <outline type="rss" text="Hacker News" xmlUrl="https://news.ycombinator.com/rss" htmlUrl="https://news.ycombinator.com"/>
    </outline>
    <outline text="News" title="News">
      <outline type="rss" text="BBC" xmlUrl="https://feeds.bbci.co.uk/news/rss.xml" htmlUrl="https://bbc.co.uk/news"/>
    </outline>
  </body>
</opml>`

func TestParseOPML(t *testing.T) {
	doc, err := ParseOPML([]byte(opmlXML))
	if err != nil {
		t.Fatalf("ParseOPML() error = %v", err)
	}

	assertEqual(t, "version", doc.Version, "2.0")
	assertEqual(t, "head title", doc.Head.Title, "My Feeds")
	assertEqual(t, "head dateCreated", doc.Head.DateCreated, "2025-01-01")
	assertEqual(t, "outline count", len(doc.Body.Outlines), 2)

	tech := doc.Body.Outlines[0]
	assertEqual(t, "group[0] text", tech.Text, "Tech")
	assertEqual(t, "group[0] feeds", len(tech.Outlines), 2)
	assertEqual(t, "group[0][0] text", tech.Outlines[0].Text, "Go Blog")
	assertEqual(t, "group[0][0] xmlUrl", tech.Outlines[0].XMLUrl, "https://go.dev/blog/feed.atom")
	assertEqual(t, "group[0][0] type", tech.Outlines[0].Type, "rss")

	news := doc.Body.Outlines[1]
	assertEqual(t, "group[1] text", news.Text, "News")
	assertEqual(t, "group[1] feeds", len(news.Outlines), 1)
}

func TestParseOPML_InvalidXML(t *testing.T) {
	_, err := ParseOPML([]byte("not xml at all"))
	if err == nil {
		t.Fatal("ParseOPML() expected error for invalid XML")
	}
}

// --- OutlineArray.AddOutline ---

func TestAddOutline_NewGroup(t *testing.T) {
	a := OutlineArray{}
	a = a.AddOutline("Tech", &Outline{Text: "Go Blog", XMLUrl: "https://go.dev/blog/feed.atom"})

	assertEqual(t, "group count", len(a), 1)
	assertEqual(t, "group text", a[0].Text, "Tech")
	assertEqual(t, "feed count", len(a[0].Outlines), 1)
	assertEqual(t, "feed text", a[0].Outlines[0].Text, "Go Blog")
}

func TestAddOutline_ExistingGroup(t *testing.T) {
	a := OutlineArray{}
	a = a.AddOutline("Tech", &Outline{Text: "Go Blog", XMLUrl: "https://go.dev/blog/feed.atom"})
	a = a.AddOutline("Tech", &Outline{Text: "Hacker News", XMLUrl: "https://news.ycombinator.com/rss"})

	assertEqual(t, "group count", len(a), 1)
	assertEqual(t, "feed count", len(a[0].Outlines), 2)
	assertEqual(t, "feed[1] text", a[0].Outlines[1].Text, "Hacker News")
}

func TestAddOutline_MultipleGroups(t *testing.T) {
	a := OutlineArray{}
	a = a.AddOutline("Tech", &Outline{Text: "Go Blog", XMLUrl: "https://go.dev/blog/feed.atom"})
	a = a.AddOutline("News", &Outline{Text: "BBC", XMLUrl: "https://feeds.bbci.co.uk/news/rss.xml"})

	assertEqual(t, "group count", len(a), 2)
	assertEqual(t, "group[0] text", a[0].Text, "Tech")
	assertEqual(t, "group[1] text", a[1].Text, "News")
}

func TestAddOutline_InvalidOutline(t *testing.T) {
	a := OutlineArray{}
	// empty text
	result := a.AddOutline("Tech", &Outline{XMLUrl: "https://example.com"})
	assertEqual(t, "empty text", len(result), 0)

	// empty xmlUrl
	result = a.AddOutline("Tech", &Outline{Text: "Test"})
	assertEqual(t, "empty xmlUrl", len(result), 0)
}

func TestAddOutline_EmptyGroupText(t *testing.T) {
	a := OutlineArray{}
	result := a.AddOutline("", &Outline{Text: "Test", XMLUrl: "https://example.com"})
	assertEqual(t, "empty group", len(result), 0)
}
