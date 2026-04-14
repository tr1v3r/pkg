package rss

import "testing"

const jsonFeedV11 = `{
  "version": "https://jsonfeed.org/version/1.1",
  "title": "My Test Feed",
  "home_page_url": "https://example.org/",
  "feed_url": "https://example.org/feed.json",
  "description": "A test JSON Feed",
  "user_comment": "Test feed for unit tests",
  "icon": "https://example.org/icon.png",
  "favicon": "https://example.org/favicon.ico",
  "authors": [{"name": "Jane Doe", "url": "https://example.org/jane", "avatar": "https://example.org/jane.png"}],
  "language": "en-US",
  "expired": false,
  "hubs": [{"type": "WebSub", "url": "https://example.org/hub"}],
  "items": [
    {
      "id": "1",
      "url": "https://example.org/post/1",
      "external_url": "https://example.org/external/1",
      "title": "First Post",
      "content_html": "<p>Hello, world!</p>",
      "summary": "A short summary",
      "image": "https://example.org/img/1.png",
      "banner_image": "https://example.org/banner/1.png",
      "date_published": "2025-01-06T00:00:00Z",
      "date_modified": "2025-01-06T12:00:00Z",
      "authors": [{"name": "Jane Doe"}],
      "tags": ["tech", "go"],
      "language": "en",
      "attachments": [
        {
          "url": "https://example.org/audio/1.mp3",
          "mime_type": "audio/mpeg",
          "title": "Podcast Episode",
          "size_in_bytes": 12345678,
          "duration_in_seconds": 987
        }
      ]
    },
    {
      "id": "2",
      "content_text": "Plain text content.",
      "date_published": "2025-01-07T00:00:00Z"
    }
  ]
}`

const jsonFeedV10 = `{
  "version": "https://jsonfeed.org/version/1",
  "title": "Legacy Feed",
  "items": [{"id": "a1", "title": "Old item"}]
}`

func TestParseJSONFeed(t *testing.T) {
	feed, err := ParseJSONFeed([]byte(jsonFeedV11))
	if err != nil {
		t.Fatalf("ParseJSONFeed() error = %v", err)
	}

	assertEqual(t, "version", feed.Version, "https://jsonfeed.org/version/1.1")
	assertEqual(t, "title", feed.Title, "My Test Feed")
	assertEqual(t, "home_page_url", feed.HomePageURL, "https://example.org/")
	assertEqual(t, "feed_url", feed.FeedURL, "https://example.org/feed.json")
	assertEqual(t, "description", feed.Description, "A test JSON Feed")
	assertEqual(t, "user_comment", feed.UserComment, "Test feed for unit tests")
	assertEqual(t, "icon", feed.Icon, "https://example.org/icon.png")
	assertEqual(t, "favicon", feed.Favicon, "https://example.org/favicon.ico")
	assertEqual(t, "language", feed.Language, "en-US")
	assertEqual(t, "expired", feed.Expired, false)

	assertEqual(t, "authors count", len(feed.Authors), 1)
	assertEqual(t, "author name", feed.Authors[0].Name, "Jane Doe")
	assertEqual(t, "author url", feed.Authors[0].URL, "https://example.org/jane")
	assertEqual(t, "author avatar", feed.Authors[0].Avatar, "https://example.org/jane.png")

	assertEqual(t, "hubs count", len(feed.Hubs), 1)
	assertEqual(t, "hub type", feed.Hubs[0].Type, "WebSub")
	assertEqual(t, "hub url", feed.Hubs[0].URL, "https://example.org/hub")

	assertEqual(t, "items count", len(feed.Items), 2)

	// Item 1 — full fields
	item1 := feed.Items[0]
	assertEqual(t, "item[0].ID", item1.ID, "1")
	assertEqual(t, "item[0].URL", item1.URL, "https://example.org/post/1")
	assertEqual(t, "item[0].ExternalURL", item1.ExternalURL, "https://example.org/external/1")
	assertEqual(t, "item[0].Title", item1.Title, "First Post")
	assertEqual(t, "item[0].ContentHTML", item1.ContentHTML, "<p>Hello, world!</p>")
	assertEqual(t, "item[0].ContentText", item1.ContentText, "")
	assertEqual(t, "item[0].Summary", item1.Summary, "A short summary")
	assertEqual(t, "item[0].Image", item1.Image, "https://example.org/img/1.png")
	assertEqual(t, "item[0].BannerImage", item1.BannerImage, "https://example.org/banner/1.png")
	assertEqual(t, "item[0].DatePublished", item1.DatePublished, "2025-01-06T00:00:00Z")
	assertEqual(t, "item[0].DateModified", item1.DateModified, "2025-01-06T12:00:00Z")
	assertEqual(t, "item[0].Language", item1.Language, "en")

	assertEqual(t, "item[0].authors count", len(item1.Authors), 1)
	assertEqual(t, "item[0].author name", item1.Authors[0].Name, "Jane Doe")

	assertEqual(t, "item[0].tags count", len(item1.Tags), 2)
	assertEqual(t, "item[0].tag[0]", item1.Tags[0], "tech")
	assertEqual(t, "item[0].tag[1]", item1.Tags[1], "go")

	assertEqual(t, "item[0].attachments count", len(item1.Attachments), 1)
	att := item1.Attachments[0]
	assertEqual(t, "attachment url", att.URL, "https://example.org/audio/1.mp3")
	assertEqual(t, "attachment mime_type", att.MimeType, "audio/mpeg")
	assertEqual(t, "attachment title", att.Title, "Podcast Episode")
	assertEqual(t, "attachment size", att.SizeInBytes, int64(12345678))
	assertEqual(t, "attachment duration", att.DurationInSeconds, int64(987))

	// Item 2 — minimal
	item2 := feed.Items[1]
	assertEqual(t, "item[1].ID", item2.ID, "2")
	assertEqual(t, "item[1].ContentText", item2.ContentText, "Plain text content.")
	assertEqual(t, "item[1].DatePublished", item2.DatePublished, "2025-01-07T00:00:00Z")
	assertEqual(t, "item[1].Title", item2.Title, "")
	assertEqual(t, "item[1].ContentHTML", item2.ContentHTML, "")
	assertEqual(t, "item[1].URL", item2.URL, "")
}

func TestParseJSONFeed_InvalidJSON(t *testing.T) {
	_, err := ParseJSONFeed([]byte("not json"))
	if err == nil {
		t.Fatal("ParseJSONFeed() expected error for invalid JSON")
	}
}

func TestParseJSONFeed_Minimal(t *testing.T) {
	data := []byte(`{"version":"https://jsonfeed.org/version/1.1","title":"Minimal","items":[{"id":"1"}]}`)
	feed, err := ParseJSONFeed(data)
	if err != nil {
		t.Fatalf("ParseJSONFeed() error = %v", err)
	}
	assertEqual(t, "version", feed.Version, "https://jsonfeed.org/version/1.1")
	assertEqual(t, "title", feed.Title, "Minimal")
	assertEqual(t, "items count", len(feed.Items), 1)
	assertEqual(t, "item id", feed.Items[0].ID, "1")
	assertEqual(t, "home_page_url", feed.HomePageURL, "")
	assertEqual(t, "authors", len(feed.Authors), 0)
	assertEqual(t, "hubs", len(feed.Hubs), 0)
}

func TestParseJSONFeed_V10(t *testing.T) {
	feed, err := ParseJSONFeed([]byte(jsonFeedV10))
	if err != nil {
		t.Fatalf("ParseJSONFeed() error = %v", err)
	}
	assertEqual(t, "version", feed.Version, "https://jsonfeed.org/version/1")
	assertEqual(t, "title", feed.Title, "Legacy Feed")
	assertEqual(t, "items count", len(feed.Items), 1)
	assertEqual(t, "item id", feed.Items[0].ID, "a1")
	assertEqual(t, "item title", feed.Items[0].Title, "Old item")
}

func TestParseJSONFeed_EmptyItems(t *testing.T) {
	data := []byte(`{"version":"https://jsonfeed.org/version/1.1","title":"Empty","items":[]}`)
	feed, err := ParseJSONFeed(data)
	if err != nil {
		t.Fatalf("ParseJSONFeed() error = %v", err)
	}
	assertEqual(t, "items count", len(feed.Items), 0)
}
