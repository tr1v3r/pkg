package rss

import "time"

// ToJSONFeed converts an RSS feed to a JSON Feed document.
func (r *RSS) ToJSONFeed() *JSONFeed {
	jf := &JSONFeed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       r.Channel.Title,
		Description: r.Channel.Description,
		HomePageURL: r.Channel.Link,
	}

	if r.Channel.Image != nil {
		jf.Icon = r.Channel.Image.URL
	}

	for _, item := range r.Channel.Items {
		ji := JSONFeedItem{
			ID:    item.GUID,
			URL:   item.Link,
			Title: item.Title,
		}

		// Content: prefer content:encoded, fall back to description.
		if item.Content != "" {
			ji.ContentHTML = item.Content
		} else {
			ji.ContentHTML = item.Description
		}

		if item.Author != "" {
			ji.Authors = []JSONFeedAuthor{{Name: item.Author}}
		}

		if item.PubDate != "" {
			ji.DatePublished = rfc822ToRFC3339(item.PubDate)
		}

		if item.Enclosure != nil {
			ji.Attachments = []JSONFeedAttachment{{
				URL:         item.Enclosure.URL,
				MimeType:    item.Enclosure.Type,
				SizeInBytes: item.Enclosure.Length,
			}}
		}

		for _, cat := range item.Categories {
			ji.Tags = append(ji.Tags, cat.Value)
		}

		jf.Items = append(jf.Items, ji)
	}

	return jf
}

// ToJSONFeed converts an Atom feed to a JSON Feed document.
func (f *Feed) ToJSONFeed() *JSONFeed {
	jf := &JSONFeed{
		Version: "https://jsonfeed.org/version/1.1",
		Title:   f.Title,
	}

	for _, l := range f.Links {
		switch l.Rel {
		case "alternate", "":
			jf.HomePageURL = l.Href
		case "self":
			jf.FeedURL = l.Href
		case "hub":
			jf.Hubs = append(jf.Hubs, JSONFeedHub{Type: "WebSub", URL: l.Href})
		}
	}

	for _, entry := range f.Entries {
		ji := JSONFeedItem{
			ID:          entry.ID,
			Title:       entry.Title,
			Summary:     entry.Summary,
			ContentHTML: entry.Content,
		}

		if entry.Published != "" {
			ji.DatePublished = entry.Published
		}
		if entry.Updated != "" {
			ji.DateModified = entry.Updated
		}

		if entry.Author.Name != "" || entry.Author.URI != "" {
			a := JSONFeedAuthor{Name: entry.Author.Name, URL: entry.Author.URI}
			ji.Authors = []JSONFeedAuthor{a}
		}

		for _, l := range entry.Links {
			switch l.Rel {
			case "alternate", "":
				if ji.URL == "" {
					ji.URL = l.Href
				}
			case "related":
				ji.ExternalURL = l.Href
			case "enclosure":
				ji.Attachments = append(ji.Attachments, JSONFeedAttachment{
					URL:      l.Href,
					MimeType: l.Type,
				})
			}
		}

		for _, cat := range entry.Categories {
			ji.Tags = append(ji.Tags, cat.Term)
		}

		jf.Items = append(jf.Items, ji)
	}

	return jf
}

// ToRSS converts a JSON Feed to an RSS feed.
func (jf *JSONFeed) ToRSS() *RSS {
	rss := &RSS{
		Channel: Channel{
			Title:       jf.Title,
			Description: jf.Description,
			Link:        jf.HomePageURL,
		},
	}

	for _, item := range jf.Items {
		ri := Item{
			Title:  item.Title,
			Link:   item.URL,
			GUID:   item.ID,
			Author: jsonFeedAuthorName(item.Authors),
		}

		// Content: prefer content_html, fall back to content_text.
		if item.ContentHTML != "" {
			ri.Content = item.ContentHTML
		}
		if item.ContentHTML == "" && item.ContentText != "" {
			ri.Description = item.ContentText
		} else if item.Summary != "" {
			ri.Description = item.Summary
		}

		if item.DatePublished != "" {
			ri.PubDate = rfc3339ToRFC822(item.DatePublished)
		}

		if len(item.Attachments) > 0 {
			ri.Enclosure = &Enclosure{
				URL:    item.Attachments[0].URL,
				Type:   item.Attachments[0].MimeType,
				Length: item.Attachments[0].SizeInBytes,
			}
		}

		for _, tag := range item.Tags {
			ri.Categories = append(ri.Categories, Category{Value: tag})
		}

		rss.Channel.Items = append(rss.Channel.Items, ri)
	}

	return rss
}

// ToAtom converts a JSON Feed to an Atom feed.
func (jf *JSONFeed) ToAtom() *Feed {
	feed := &Feed{Title: jf.Title}

	if jf.HomePageURL != "" {
		feed.Links = append(feed.Links, Link{Href: jf.HomePageURL, Rel: "alternate"})
	}
	if jf.FeedURL != "" {
		feed.Links = append(feed.Links, Link{Href: jf.FeedURL, Rel: "self"})
	}
	for _, hub := range jf.Hubs {
		feed.Links = append(feed.Links, Link{Href: hub.URL, Rel: "hub"})
	}

	for _, item := range jf.Items {
		entry := Entry{
			Title:     item.Title,
			ID:        item.ID,
			Summary:   item.Summary,
			Content:   item.ContentHTML,
			Published: item.DatePublished,
			Updated:   item.DateModified,
		}

		if entry.Content == "" {
			entry.Content = item.ContentText
		}

		if len(item.Authors) > 0 {
			entry.Author = Author{
				Name: item.Authors[0].Name,
				URI:  item.Authors[0].URL,
			}
		}

		if item.URL != "" {
			entry.Links = append(entry.Links, Link{Href: item.URL, Rel: "alternate"})
		}
		if item.ExternalURL != "" {
			entry.Links = append(entry.Links, Link{Href: item.ExternalURL, Rel: "related"})
		}
		for _, att := range item.Attachments {
			entry.Links = append(entry.Links, Link{
				Href: att.URL,
				Rel:  "enclosure",
				Type: att.MimeType,
			})
		}

		for _, tag := range item.Tags {
			entry.Categories = append(entry.Categories, AtomCategory{Term: tag})
		}

		feed.Entries = append(feed.Entries, entry)
	}

	return feed
}

// jsonFeedAuthorName returns the first author's name, or empty string.
func jsonFeedAuthorName(authors []JSONFeedAuthor) string {
	if len(authors) > 0 {
		return authors[0].Name
	}
	return ""
}

// rfc822ToRFC3339 converts an RFC 822 date string to RFC 3339 format.
func rfc822ToRFC3339(s string) string {
	t, err := time.Parse(time.RFC1123Z, s)
	if err != nil {
		// Try without timezone name.
		t, err = time.Parse(time.RFC1123, s)
		if err != nil {
			return s // Return as-is if parsing fails.
		}
	}
	return t.Format(time.RFC3339)
}

// rfc3339ToRFC822 converts an RFC 3339 date string to RFC 822 format.
func rfc3339ToRFC822(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s // Return as-is if parsing fails.
	}
	return t.Format(time.RFC1123Z)
}
