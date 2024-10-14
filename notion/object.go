package notion

import "encoding/json"

// https://developers.notion.com/reference/request-limits
const rateLimit = 3

// Object notion object
type Object struct {
	PureObject
	CreatedTime    string              `json:"created_time"`
	CreatedBy      PureObject          `json:"created_by"`
	LastEditedTime string              `json:"last_edited_time"`
	LastEditedBy   PureObject          `json:"last_edited_by"`
	Cover          FileItem            `json:"cover,omitempty"`
	Icon           IconItem            `json:"icon,omitempty"`
	Title          []TextItem          `json:"title,omitempty"`
	Description    []TextItem          `json:"description,omitempty"`
	IsInline       bool                `json:"is_inline,omitempty"`
	Properties     map[string]Property `json:"properties,omitempty"`
	Parent         PageItem            `json:"parent,omitempty"`
	URL            string              `json:"url,omitempty"`
	Archived       bool                `json:"archived,omitempty"`
	Results        []Object            `json:"results,omitempty"`
	NextCursor     string              `json:"next_cursor"`
	HasMore        bool                `json:"has_more"`
	Type           string              `json:"type"`

	Status  int    `json:"status,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// PureObject pure notion object
type PureObject struct {
	Object string `json:"object"`
	ID     string `json:"id"`
}

type PageItem struct {
	Type       string `json:"type,omitempty"`
	PageID     string `json:"page_id,omitempty"`
	DatabaseID string `json:"database_id,omitempty"`
}

type TextObjectArray []TextObject

func (a TextObjectArray) JSON() json.RawMessage {
	data, _ := json.Marshal(a)
	return data
}

type TextObject struct {
	Type        string      `json:"type,omitempty"`
	Text        TextItem    `json:"text"`
	Annotations *Annotation `json:"annotations,omitempty"`
	PlainText   string      `json:"plain_text,omitempty"`
	Href        *string     `json:"href,omitempty"`
}

type TextItem struct {
	Content string  `json:"content"`
	Link    *string `json:"link,omitempty"`
}

type Annotation struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

type FileItemArray []FileItem

func (a FileItemArray) JSON() json.RawMessage {
	data, _ := json.Marshal(a)
	return data
}

type FileItem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	External struct {
		URL string `json:"url"`
	} `json:"external"`
}

type IconItem struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

type NumberProperty struct {
	Format string `json:"format"`
}

type SelectProperty struct {
	Options []SelectOption `json:"options"`
}

type SelectOption struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}
