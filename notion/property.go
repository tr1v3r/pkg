package notion

import (
	"encoding/json"
)

// PropertyType property type
type PropertyType string

const (
	TitleProp       PropertyType = "title"
	NumberProp      PropertyType = "number"
	RichTextProp    PropertyType = "rich_text"
	SelectProp      PropertyType = "select"
	MultiSelectProp PropertyType = "multi_select"
	FilesProp       PropertyType = "files"
	URLProp         PropertyType = "url"
	DateProp        PropertyType = "date"
	CheckboxProp    PropertyType = "checkbox"
)

// Property
type Property struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	Type PropertyType `json:"type"`

	// https://developers.notion.com/reference/page-property-values#type-objects
	Date        json.RawMessage `json:"date,omitempty"`
	Title       json.RawMessage `json:"title,omitempty"`
	Number      any             `json:"number,omitempty"`
	RichText    json.RawMessage `json:"rich_text,omitempty"`
	Select      json.RawMessage `json:"select,omitempty"`
	MultiSelect json.RawMessage `json:"multi_select,omitempty"`
	Files       json.RawMessage `json:"files,omitempty"`
	URL         json.RawMessage `json:"url,omitempty"`
	Checkbox    *bool           `json:"checkbox,omitempty"`
}

// ForUpdate return update format data
func (p Property) ForUpdate() (data json.RawMessage) {
	switch {
	case p.Date != nil:
		data, _ = json.Marshal(map[PropertyType]any{DateProp: p.Date})
	case p.Title != nil:
		data, _ = json.Marshal(map[PropertyType]any{TitleProp: p.Title})
	case p.RichText != nil:
		data, _ = json.Marshal(map[PropertyType]any{RichTextProp: p.RichText})
	case p.Number != nil:
		data, _ = json.Marshal(map[PropertyType]any{NumberProp: p.Number})
	case p.Select != nil:
		data, _ = json.Marshal(map[PropertyType]any{SelectProp: p.MultiSelect})
	case p.MultiSelect != nil:
		data, _ = json.Marshal(map[PropertyType]any{MultiSelectProp: p.Select})
	case p.Files != nil:
		data, _ = json.Marshal(map[PropertyType]any{FilesProp: p.Files})
	case p.URL != nil:
		data, _ = json.Marshal(map[PropertyType]any{URLProp: p.URL})
	case p.Checkbox != nil:
		data, _ = json.Marshal(map[PropertyType]any{CheckboxProp: p.Checkbox})
	}
	return data
}

// PlainText
func (p Property) PlainText() (text string) {
	switch p.Type {
	case RichTextProp:
		if p.RichText == nil {
			return ""
		}
		texts := make([]TextObject, 0, 4)
		if err := json.Unmarshal(p.RichText, &texts); err != nil {
			return ""
		}
		for _, t := range texts {
			text += t.PlainText
		}
		return text
	case TitleProp:
		if p.Title == nil {
			return ""
		}
		texts := make([]TextObject, 0, 4)
		if err := json.Unmarshal(p.Title, &texts); err != nil {
			return ""
		}

		for _, t := range texts {
			text += t.PlainText
		}
		return text
	case SelectProp:
		if p.Select == nil {
			return ""
		}
		var sel SelectOption
		if err := json.Unmarshal(p.Select, &sel); err != nil {
			return ""
		}
		return sel.Name
	case URLProp:
		if p.URL == nil {
			return ""
		}
		var url string
		if err := json.Unmarshal(p.URL, &url); err != nil {
			return ""
		}
		return url
	default:
		return ""
	}
}

type PropertyArray []*Property

func (pa PropertyArray) ForUpdate() json.RawMessage {
	var m = make(map[string]json.RawMessage, len(pa))
	for _, p := range pa {
		m[p.Name] = p.ForUpdate()
	}
	data, _ := json.Marshal(m)
	return data
}
