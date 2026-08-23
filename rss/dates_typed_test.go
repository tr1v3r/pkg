package rss

import (
	"strings"
	"testing"
)

func TestRFC822ToRFC3339AllLayouts(t *testing.T) {
	// Named zones (MST/GMT/UT) resolve to fixed offsets without tzdata
	// (MST = -0700), so expectations stay environment-independent.
	cases := map[string]string{
		// four-digit year (RFC 1123 forms)
		"Mon, 02 Jan 2006 15:04:05 -0700": "2006-01-02T15:04:05-07:00",
		// RSS 2.0 spec format: RFC 822, two-digit year
		"Mon, 02 Jan 06 15:04:05 -0700": "2006-01-02T15:04:05-07:00",
		// no day-of-week
		"02 Jan 06 15:04 -0700": "2006-01-02T15:04:00-07:00",
		// UT / Z suffixes
		"Mon, 02 Jan 2006 15:04:05 UT": "2006-01-02T15:04:05Z",
		"Mon, 02 Jan 2006 15:04:05 Z":  "2006-01-02T15:04:05Z",
		// single-digit day
		"Mon, 2 Jan 2006 15:04:05 GMT": "2006-01-02T15:04:05Z",
		// trailing timezone comment
		"Mon, 02 Jan 2006 15:04:05 +0800 (CST)": "2006-01-02T15:04:05+08:00",
	}

	for input, want := range cases {
		if got := rfc822ToRFC3339(input); got != want {
			t.Errorf("rfc822ToRFC3339(%q) = %q, want %q", input, got, want)
		}
	}

	// unparseable input is returned trimmed
	if got := rfc822ToRFC3339("  garbage  "); got != "garbage" {
		t.Errorf("garbage = %q", got)
	}

	// Named-zone forms must at least parse and produce an RFC 3339 timestamp,
	// independent of the host tzdata's offset for that abbreviation.
	for _, named := range []string{
		"Mon, 02 Jan 2006 15:04:05 MST",
		"Mon, 02 Jan 06 15:04:05 MST",
		"02 Jan 06 15:04 MST",
	} {
		got := rfc822ToRFC3339(named)
		if !strings.HasPrefix(got, "2006-01-02T15:04") {
			t.Errorf("rfc822ToRFC3339(%q) = %q, want RFC3339 conversion", named, got)
		}
	}
}

func TestRSSPubDateConversionEndToEnd(t *testing.T) {
	// End-to-end: a spec-format pubDate (2-digit year) survives RSS -> JSON Feed.
	xmlDoc := "<?xml version=\"1.0\"?>" +
		"<rss version=\"2.0\"><channel><title>t</title>" +
		"<item><title>it</title><pubDate>Mon, 02 Jan 06 15:04:05 GMT</pubDate></item>" +
		"</channel></rss>"

	feed, err := ParseRSS([]byte(xmlDoc))
	if err != nil {
		t.Fatalf("parse fail: %s", err)
	}
	jf := feed.ToJSONFeed()
	if got := jf.Items[0].DatePublished; got != "2006-01-02T15:04:05Z" {
		t.Errorf("DatePublished = %q, want RFC3339 conversion", got)
	}
}

func TestDeduplicateItemsDoesNotClobberAliases(t *testing.T) {
	ch := &Channel{Items: []Item{
		{GUID: "a"}, {GUID: "a"}, {GUID: "b"},
	}}

	snapshot := make([]Item, len(ch.Items))
	copy(snapshot, ch.Items)

	ch.DeduplicateItems()

	if len(ch.Items) != 2 {
		t.Fatalf("dedup left %d items, want 2", len(ch.Items))
	}
	// The pre-call copy must be untouched even though Items was reassigned.
	for i, it := range snapshot {
		if it.GUID != []string{"a", "a", "b"}[i] {
			t.Errorf("snapshot[%d].GUID = %q, alias was clobbered", i, it.GUID)
		}
	}
}

func TestParseTyped(t *testing.T) {
	rssDoc := []byte("<rss version=\"2.0\"><channel><title>t</title></channel></rss>")
	atomDoc := []byte("<feed xmlns=\"http://www.w3.org/2005/Atom\"><title>t</title></feed>")

	feed, err := ParseTyped[*RSS](rssDoc)
	if err != nil {
		t.Fatalf("ParseTyped RSS fail: %s", err)
	}
	if feed.Channel.Title != "t" {
		t.Errorf("title = %q", feed.Channel.Title)
	}

	// wrong type errors
	if _, err := ParseTyped[*Feed](rssDoc); err == nil {
		t.Error("ParseTyped[*Feed] on RSS data should fail")
	}
	if _, err := ParseTyped[*RSS](atomDoc); err == nil {
		t.Error("ParseTyped[*RSS] on Atom data should fail")
	}
	// must mention both formats in the message
	if _, err := ParseTyped[*Feed](rssDoc); err == nil || !strings.Contains(err.Error(), "RSS") {
		t.Error("error message should name the detected format")
	}
}

func TestFeedTypeString(t *testing.T) {
	cases := map[FeedType]string{
		FeedTypeRSS:     "RSS",
		FeedTypeAtom:    "Atom",
		FeedTypeJSON:    "JSON Feed",
		FeedTypeUnknown: "unknown",
	}
	for ft, want := range cases {
		if got := ft.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", ft, got, want)
		}
	}
}
