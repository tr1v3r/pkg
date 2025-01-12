package notion

import (
	"encoding/json"
)

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
	RequestID      string              `json:"request_id,omitempty"`
	PropertyItem   Property            `json:"property,omitempty"`

	Relation RelationItem `json:"relation,omitempty"`
	RichText TextObject   `json:"rich_text,omitempty"`

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

// DateObject ...
// https://developers.notion.com/reference/page-property-values#date
type DateObject struct {
	Start    string `json:"start"`         // ISO 8601 date and time
	End      string `json:"end,omitempty"` // ISO 8601 date and time
	TimeZone string `json:"time_zone,omitempty"`
}

func (o DateObject) JSON() json.RawMessage {
	data, _ := json.Marshal(o)
	return data
}

type RelationItem struct {
	ID string `json:"id"`
}

type RelationObject []RelationItem

func (o RelationObject) IDs() (ids []string) {
	for _, item := range o {
		ids = append(ids, item.ID)
	}
	return ids
}

func (o RelationObject) JSON() json.RawMessage {
	data, _ := json.Marshal(o)
	return data
}

// RollupObject cannot be used when update
type RollupObject struct {
	Type     string `json:"type"`
	Function string `json:"function"`
	// array || date || incomplete || number || unsupported
	Number int `json:"number,omitempty"`
	Array  []struct {
		Type     string       `json:"type"`
		RichText []TextObject `json:"rich_text,omitempty"`
	} `json:"array,omitempty"`
}

func (o RollupObject) PlainStrings() (strs []string) {
	if o.Type != "array" || len(o.Array) == 0 {
		return nil
	}
	for _, item := range o.Array {
		if item.Type != "rich_text" {
			continue
		}
		for _, text := range item.RichText {
			strs = append(strs, text.PlainText)
		}
	}
	return strs
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
	Options []SelectOptionObject `json:"options"`
}

type SelectOptionObject struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

func (o SelectOptionObject) JSON() json.RawMessage {
	data, _ := json.Marshal(o)
	return data
}
