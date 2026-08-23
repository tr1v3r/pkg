package notion

import (
	"encoding/json"
)

// https://developers.notion.com/reference/request-limits
const rateLimit = 3

// Object types
const (
	ErrorObjectType = "error"
)

// Object notion object
type Object struct {
	PureObject
	CreatedTime    string              `json:"created_time"`
	CreatedBy      PureObject          `json:"created_by"`
	LastEditedTime string              `json:"last_edited_time"`
	LastEditedBy   PureObject          `json:"last_edited_by"`
	Cover          FileItem            `json:"cover"`
	Icon           IconItem            `json:"icon"`
	Title          []TextItem          `json:"title,omitempty"`
	Description    []TextItem          `json:"description,omitempty"`
	IsInline       bool                `json:"is_inline,omitempty"`
	Properties     map[string]Property `json:"properties,omitempty"`
	Parent         PageItem            `json:"parent"`
	URL            string              `json:"url,omitempty"`
	Archived       bool                `json:"archived,omitempty"`
	Results        []Object            `json:"results,omitempty"`
	NextCursor     string              `json:"next_cursor"`
	HasMore        bool                `json:"has_more"`
	Type           string              `json:"type"`
	RequestID      string              `json:"request_id,omitempty"`
	PropertyItem   Property            `json:"property"`

	Relation RelationItem `json:"relation"`
	RichText TextObject   `json:"rich_text"`

	Status  int    `json:"status,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// PureObject pure notion object
type PureObject struct {
	Object string `json:"object"`
	ID     string `json:"id"`
}

// PageItem represents the corresponding API object.
type PageItem struct {
	Type       string `json:"type,omitempty"`
	PageID     string `json:"page_id,omitempty"`
	DatabaseID string `json:"database_id,omitempty"`
}

// TextObjectArray represents the corresponding API object.
type TextObjectArray []TextObject

// JSON returns the value serialized as json.RawMessage.
func (a TextObjectArray) JSON() json.RawMessage {
	data, _ := json.Marshal(a)
	return data
}

// TextObject represents the corresponding API object.
type TextObject struct {
	Type        string      `json:"type,omitempty"`
	Text        TextItem    `json:"text"`
	Annotations *Annotation `json:"annotations,omitempty"`
	PlainText   string      `json:"plain_text,omitempty"`
	Href        *string     `json:"href,omitempty"`
}

// TextItem represents the corresponding API object.
type TextItem struct {
	Content string  `json:"content"`
	Link    *string `json:"link,omitempty"`
}

// Annotation carries rich-text styling flags.
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

// JSON returns the value serialized as json.RawMessage.
func (o DateObject) JSON() json.RawMessage {
	data, _ := json.Marshal(o)
	return data
}

// RelationItem represents the corresponding API object.
type RelationItem struct {
	ID string `json:"id"`
}

// RelationObject represents the corresponding API object.
type RelationObject []RelationItem

// IDs returns the identifiers of every item.
func (o RelationObject) IDs() (ids []string) {
	for _, item := range o {
		ids = append(ids, item.ID)
	}
	return ids
}

// JSON returns the value serialized as json.RawMessage.
func (o RelationObject) JSON() json.RawMessage {
	data, _ := json.Marshal(o)
	return data
}

// rollupTypeArray is the "array" rollup type value.
const rollupTypeArray = "array"

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

// PlainStrings extracts the plain-text representation.
func (o RollupObject) PlainStrings() (strs []string) {
	if o.Type != rollupTypeArray || len(o.Array) == 0 {
		return nil
	}
	for _, item := range o.Array {
		if item.Type != string(RichTextProp) {
			continue
		}
		for _, text := range item.RichText {
			strs = append(strs, text.PlainText)
		}
	}
	return strs
}

// FileItemArray represents the corresponding API object.
type FileItemArray []FileItem

// JSON returns the value serialized as json.RawMessage.
func (a FileItemArray) JSON() json.RawMessage {
	data, _ := json.Marshal(a)
	return data
}

// FileItem represents the corresponding API object.
type FileItem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	External struct {
		URL string `json:"url"`
	} `json:"external"`
}

// IconItem represents the corresponding API object.
type IconItem struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

// NumberProperty is a number-typed page property value.
type NumberProperty struct {
	Format string `json:"format"`
}

// SelectProperty is a select-typed page property value.
type SelectProperty struct {
	Options []SelectOptionObject `json:"options"`
}

// SelectOptionObject represents the corresponding API object.
type SelectOptionObject struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// JSON returns the value serialized as json.RawMessage.
func (o SelectOptionObject) JSON() json.RawMessage {
	data, _ := json.Marshal(o)
	return data
}
