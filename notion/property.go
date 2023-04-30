package notion

import "encoding/json"

// PropertyType property type
type PropertyType string

const (
	TitleProp       PropertyType = "title"
	NumberProp      PropertyType = "number"
	RichTextProp    PropertyType = "rich_text"
	SelectProp      PropertyType = "select"
	MultiSelectProp PropertyType = "multi_select"
	FilesProp       PropertyType = "files"
)

// Property
type Property struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	Type PropertyType `json:"type"`

	Title       interface{} `json:"title,omitempty"`
	Number      interface{} `json:"number,omitempty"`
	RichText    interface{} `json:"rich_text,omitempty"`
	Select      interface{} `json:"select,omitempty"`
	MultiSelect interface{} `json:"multi_select,omitempty"`
	Files       interface{} `json:"files,omitempty"`
}

// ForUpdate return update format data
func (p *Property) ForUpdate() map[PropertyType]interface{} {
	switch {
	case p.Title != nil:
		return map[PropertyType]interface{}{TitleProp: p.Title}
	case p.RichText != nil:
		return map[PropertyType]interface{}{RichTextProp: p.RichText}
	case p.Number != nil:
		return map[PropertyType]interface{}{NumberProp: p.Number}
	case p.Select != nil:
		return map[PropertyType]interface{}{SelectProp: p.MultiSelect}
	case p.MultiSelect != nil:
		return map[PropertyType]interface{}{MultiSelectProp: p.Select}
	case p.Files != nil:
		return map[PropertyType]interface{}{FilesProp: p.Files}
	default:
		return nil
	}
}

// PlainText parse rich text to plain text
func (p *Property) PlainText() (text string) {
	if p.Type != RichTextProp || p.RichText == nil {
		return ""
	}

	data, _ := json.Marshal(p.RichText)
	texts := make([]TextObject, 0, 4)
	if err := json.Unmarshal(data, &texts); err != nil {
		return ""
	}

	for _, t := range texts {
		text += t.PlainText
	}
	return text
}

type PropertyArray []*Property

func (pa PropertyArray) ForUpdate() map[string]interface{} {
	var m = make(map[string]interface{}, len(pa))
	for _, p := range pa {
		m[p.Name] = p.ForUpdate()
	}
	return m
}
