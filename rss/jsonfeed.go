package rss

// JSONFeed represents a JSON Feed document (version 1.0 or 1.1).
// See https://www.jsonfeed.org/version/1.1/
type JSONFeed struct {
	Version     string           `json:"version"`
	Title       string           `json:"title"`
	Items       []JSONFeedItem   `json:"items"`
	HomePageURL string           `json:"home_page_url,omitempty"`
	FeedURL     string           `json:"feed_url,omitempty"`
	Description string           `json:"description,omitempty"`
	UserComment string           `json:"user_comment,omitempty"`
	NextURL     string           `json:"next_url,omitempty"`
	Icon        string           `json:"icon,omitempty"`
	Favicon     string           `json:"favicon,omitempty"`
	Authors     []JSONFeedAuthor `json:"authors,omitempty"`
	Language    string           `json:"language,omitempty"`
	Expired     bool             `json:"expired,omitempty"`
	Hubs        []JSONFeedHub    `json:"hubs,omitempty"`
}

// JSONFeedItem represents a single item in a JSON Feed.
type JSONFeedItem struct {
	ID            string               `json:"id"`
	URL           string               `json:"url,omitempty"`
	ExternalURL   string               `json:"external_url,omitempty"`
	Title         string               `json:"title,omitempty"`
	ContentHTML   string               `json:"content_html,omitempty"`
	ContentText   string               `json:"content_text,omitempty"`
	Summary       string               `json:"summary,omitempty"`
	Image         string               `json:"image,omitempty"`
	BannerImage   string               `json:"banner_image,omitempty"`
	DatePublished string               `json:"date_published,omitempty"`
	DateModified  string               `json:"date_modified,omitempty"`
	Authors       []JSONFeedAuthor     `json:"authors,omitempty"`
	Tags          []string             `json:"tags,omitempty"`
	Language      string               `json:"language,omitempty"`
	Attachments   []JSONFeedAttachment `json:"attachments,omitempty"`
}

// JSONFeedAuthor represents an author in a JSON Feed.
type JSONFeedAuthor struct {
	Name   string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

// JSONFeedAttachment represents an attachment in a JSON Feed item.
type JSONFeedAttachment struct {
	URL               string `json:"url"`
	MimeType          string `json:"mime_type"`
	Title             string `json:"title,omitempty"`
	SizeInBytes       int64  `json:"size_in_bytes,omitempty"`
	DurationInSeconds int64  `json:"duration_in_seconds,omitempty"`
}

// JSONFeedHub represents a hub endpoint for real-time notifications.
type JSONFeedHub struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}
