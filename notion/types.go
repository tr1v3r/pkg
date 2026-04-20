package notion

import "encoding/json"

// Database represents a Notion database object.
type Database struct {
	Object         string             `json:"object"`
	ID             string             `json:"id"`
	CreatedTime    string             `json:"created_time"`
	CreatedBy      UserRef            `json:"created_by"`
	LastEditedTime string             `json:"last_edited_time"`
	LastEditedBy   UserRef            `json:"last_edited_by"`
	Title          []TextObject       `json:"title,omitempty"`
	Description    []TextObject       `json:"description,omitempty"`
	IsInline       bool               `json:"is_inline,omitempty"`
	Properties     map[string]Property `json:"properties,omitempty"`
	Parent         ParentRef          `json:"parent,omitempty"`
	URL            string             `json:"url,omitempty"`
	Icon           *IconItem          `json:"icon,omitempty"`
	Cover          *FileItem          `json:"cover,omitempty"`
}

// Page represents a Notion page object.
type Page struct {
	Object         string             `json:"object"`
	ID             string             `json:"id"`
	CreatedTime    string             `json:"created_time"`
	CreatedBy      UserRef            `json:"created_by"`
	LastEditedTime string             `json:"last_edited_time"`
	LastEditedBy   UserRef            `json:"last_edited_by"`
	Parent         ParentRef          `json:"parent,omitempty"`
	Properties     map[string]Property `json:"properties,omitempty"`
	URL            string             `json:"url,omitempty"`
	Archived       bool               `json:"archived,omitempty"`
	InTrash        bool               `json:"in_trash,omitempty"`
	Icon           *IconItem          `json:"icon,omitempty"`
	Cover          *FileItem          `json:"cover,omitempty"`
}

// Block represents a Notion block object.
type Block struct {
	Object         string          `json:"object"`
	ID             string          `json:"id"`
	Parent         ParentRef       `json:"parent,omitempty"`
	Type           string          `json:"type"`
	CreatedTime    string          `json:"created_time"`
	CreatedBy      UserRef         `json:"created_by"`
	LastEditedTime string          `json:"last_edited_time"`
	LastEditedBy   UserRef         `json:"last_edited_by"`
	HasChildren    bool            `json:"has_children"`
	InTrash        bool            `json:"in_trash,omitempty"`

	Paragraph        *RichTextBlock `json:"paragraph,omitempty"`
	Heading1         *RichTextBlock `json:"heading_1,omitempty"`
	Heading2         *RichTextBlock `json:"heading_2,omitempty"`
	Heading3         *RichTextBlock `json:"heading_3,omitempty"`
	BulletedListItem *RichTextBlock `json:"bulleted_list_item,omitempty"`
	NumberedListItem *RichTextBlock `json:"numbered_list_item,omitempty"`
	ToDo             *ToDoBlock     `json:"to_do,omitempty"`
	Toggle           *RichTextBlock `json:"toggle,omitempty"`
	Code             *CodeBlock     `json:"code,omitempty"`
	Quote            *RichTextBlock `json:"quote,omitempty"`
	Callout          *CalloutBlock  `json:"callout,omitempty"`
	Divider          *struct{}      `json:"divider,omitempty"`
	Embed            *EmbedBlock    `json:"embed,omitempty"`
	Image            *FileBlock     `json:"image,omitempty"`
	Video            *FileBlock     `json:"video,omitempty"`
	Bookmark         *BookmarkBlock `json:"bookmark,omitempty"`
	TableOfContents  *struct {
		Color string `json:"color,omitempty"`
	} `json:"table_of_contents,omitempty"`
	ChildPage      *ChildPageBlock `json:"child_page,omitempty"`
	ChildDatabase  *ChildDBBlock   `json:"child_database,omitempty"`
}

// RichTextBlock is a block that contains rich text content.
type RichTextBlock struct {
	RichText []TextObject `json:"rich_text"`
	Color    string       `json:"color,omitempty"`
	IsToggleable bool     `json:"is_toggleable,omitempty"`
}

// ToDoBlock is a checkbox block.
type ToDoBlock struct {
	RichText []TextObject `json:"rich_text"`
	Checked  bool         `json:"checked"`
	Color    string       `json:"color,omitempty"`
}

// CodeBlock is a code block with language.
type CodeBlock struct {
	RichText []TextObject `json:"rich_text"`
	Language string       `json:"language"`
}

// CalloutBlock is a callout block with icon.
type CalloutBlock struct {
	RichText []TextObject `json:"rich_text"`
	Icon     *IconItem    `json:"icon,omitempty"`
	Color    string       `json:"color,omitempty"`
}

// EmbedBlock is an embedded URL block.
type EmbedBlock struct {
	URL string `json:"url"`
}

// FileBlock represents an image, video, audio, or file block.
type FileBlock struct {
	Type     string       `json:"type"`
	External *FileRef     `json:"external,omitempty"`
	File     *HostedFile  `json:"file,omitempty"`
	Caption  []TextObject `json:"caption,omitempty"`
}

// FileRef is an external file reference.
type FileRef struct {
	URL string `json:"url"`
}

// HostedFile is a Notion-hosted file with expiry.
type HostedFile struct {
	URL        string `json:"url"`
	ExpiryTime string `json:"expiry_time"`
}

// BookmarkBlock is a bookmark block.
type BookmarkBlock struct {
	URL     string       `json:"url"`
	Caption []TextObject `json:"caption,omitempty"`
}

// ChildPageBlock represents a nested page reference.
type ChildPageBlock struct {
	Title string `json:"title"`
}

// ChildDBBlock represents a nested database reference.
type ChildDBBlock struct {
	Title string `json:"title"`
}

// User represents a Notion user.
type User struct {
	Object    string `json:"object"`
	ID        string `json:"id"`
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Email     string `json:"email,omitempty"`
}

// UserRef is a lightweight user reference in created_by/last_edited_by fields.
type UserRef struct {
	Object string `json:"object"`
	ID     string `json:"id"`
}

// Comment represents a Notion comment.
type Comment struct {
	Object         string       `json:"object"`
	ID             string       `json:"id"`
	Parent         ParentRef    `json:"parent,omitempty"`
	DiscussionID   string       `json:"discussion_id,omitempty"`
	CreatedTime    string       `json:"created_time"`
	LastEditedTime string       `json:"last_edited_time,omitempty"`
	CreatedBy      UserRef      `json:"created_by"`
	RichText       []TextObject `json:"rich_text"`
}

// ParentRef references a parent resource (page, database, or workspace).
type ParentRef = PageItem

// ListResponse is a generic paginated response.
type ListResponse[T any] struct {
	Object     string `json:"object"`
	Results    []T    `json:"results"`
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
	Type       string `json:"type,omitempty"`
}

// SearchResult is a result from the Notion search API.
type SearchResult = json.RawMessage

// SearchFilter filters search results.
type SearchFilter struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

// SearchSort sorts search results.
type SearchSort struct {
	Direction string `json:"direction"`
	Timestamp string `json:"timestamp"`
}
