package notion

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProperty_ForUpdate_Date(t *testing.T) {
	dateJSON := DateObject{Start: "2024-01-15", End: "2024-01-20"}.JSON()
	p := Property{Name: "Due Date", Type: DateProp, Date: dateJSON}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(DateProp))

	var dateObj DateObject
	err = json.Unmarshal(result[string(DateProp)], &dateObj)
	assert.NoError(t, err)
	assert.Equal(t, "2024-01-15", dateObj.Start)
	assert.Equal(t, "2024-01-20", dateObj.End)
}

func TestProperty_ForUpdate_Title(t *testing.T) {
	titleJSON := TextObjectArray{
		{Text: TextItem{Content: "Hello"}, PlainText: "Hello"},
	}.JSON()
	p := Property{Name: "Name", Type: TitleProp, Title: titleJSON}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(TitleProp))
}

func TestProperty_ForUpdate_RichText(t *testing.T) {
	rtJSON := TextObjectArray{
		{Text: TextItem{Content: "world"}, PlainText: "world"},
	}.JSON()
	p := Property{Name: "Desc", Type: RichTextProp, RichText: rtJSON}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(RichTextProp))
}

func TestProperty_ForUpdate_Number(t *testing.T) {
	p := Property{Name: "Count", Type: NumberProp, Number: 42}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]any
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(NumberProp))
	assert.Equal(t, float64(42), result[string(NumberProp)])
}

func TestProperty_ForUpdate_Select(t *testing.T) {
	selJSON := SelectOptionObject{Name: "High", Color: "red"}.JSON()
	p := Property{Name: "Priority", Type: SelectProp, Select: selJSON}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(SelectProp))

	var selObj SelectOptionObject
	err = json.Unmarshal(result[string(SelectProp)], &selObj)
	assert.NoError(t, err)
	assert.Equal(t, "High", selObj.Name)
}

func TestProperty_ForUpdate_MultiSelect(t *testing.T) {
	multiJSON, _ := json.Marshal([]SelectOptionObject{
		{Name: "Go"},
		{Name: "Python"},
	})
	p := Property{Name: "Tags", Type: MultiSelectProp, MultiSelect: multiJSON}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(MultiSelectProp))

	var options []SelectOptionObject
	err = json.Unmarshal(result[string(MultiSelectProp)], &options)
	assert.NoError(t, err)
	assert.Len(t, options, 2)
	assert.Equal(t, "Go", options[0].Name)
	assert.Equal(t, "Python", options[1].Name)
}

func TestProperty_ForUpdate_Checkbox(t *testing.T) {
	checked := true
	p := Property{Name: "Done", Type: CheckboxProp, Checkbox: &checked}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]any
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(CheckboxProp))
	assert.Equal(t, true, result[string(CheckboxProp)])
}

func TestProperty_ForUpdate_URL(t *testing.T) {
	urlJSON, _ := json.Marshal("https://example.com")
	p := Property{Name: "Link", Type: URLProp, URL: urlJSON}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(URLProp))

	// Verify no double-serialization: the URL value should be a string, not a string within a string
	var urlStr string
	err = json.Unmarshal(result[string(URLProp)], &urlStr)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", urlStr)
}

func TestProperty_ForUpdate_Relation(t *testing.T) {
	relJSON := RelationObject{{ID: "abc-123"}, {ID: "def-456"}}.JSON()
	p := Property{Name: "Related", Type: RelationProp, Relation: relJSON}

	data := p.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, string(RelationProp))
}

func TestProperty_PlainText_RichText(t *testing.T) {
	rtJSON := TextObjectArray{
		{PlainText: "Hello "},
		{PlainText: "World"},
	}.JSON()
	p := Property{Type: RichTextProp, RichText: rtJSON}

	assert.Equal(t, "Hello World", p.PlainText())
}

func TestProperty_PlainText_Title(t *testing.T) {
	titleJSON := TextObjectArray{
		{PlainText: "My Page Title"},
	}.JSON()
	p := Property{Type: TitleProp, Title: titleJSON}

	assert.Equal(t, "My Page Title", p.PlainText())
}

func TestProperty_PlainText_Select(t *testing.T) {
	selJSON := SelectOptionObject{Name: "Medium"}.JSON()
	p := Property{Type: SelectProp, Select: selJSON}

	assert.Equal(t, "Medium", p.PlainText())
}

func TestProperty_PlainText_URL(t *testing.T) {
	urlJSON, _ := json.Marshal("https://example.com/page")
	p := Property{Type: URLProp, URL: urlJSON}

	assert.Equal(t, "https://example.com/page", p.PlainText())
}

func TestProperty_PlainText_Empty(t *testing.T) {
	// Nil rich text returns empty
	p1 := Property{Type: RichTextProp, RichText: nil}
	assert.Equal(t, "", p1.PlainText())

	// Nil title returns empty
	p2 := Property{Type: TitleProp, Title: nil}
	assert.Equal(t, "", p2.PlainText())

	// Unsupported type returns empty
	p3 := Property{Type: FilesProp}
	assert.Equal(t, "", p3.PlainText())

	// Nil select returns empty
	p4 := Property{Type: SelectProp, Select: nil}
	assert.Equal(t, "", p4.PlainText())
}

func TestProperty_GetRelationIDs(t *testing.T) {
	relJSON := RelationObject{{ID: "abc-123"}, {ID: "def-456"}, {ID: "ghi-789"}}.JSON()
	p := Property{Type: RelationProp, Relation: relJSON}

	ids := p.GetRelationIDs()
	assert.Equal(t, []string{"abc-123", "def-456", "ghi-789"}, ids)
}

func TestProperty_GetRelationIDs_Nil(t *testing.T) {
	p := Property{Type: RelationProp, Relation: nil}
	assert.Nil(t, p.GetRelationIDs())
}

func TestPropertyArray_ForUpdate(t *testing.T) {
	titleJSON := TextObjectArray{{PlainText: "Test Title"}}.JSON()
	checked := false
	urlJSON, _ := json.Marshal("https://example.org")

	pa := PropertyArray{
		{Name: "Title", Type: TitleProp, Title: titleJSON},
		{Name: "Done", Type: CheckboxProp, Checkbox: &checked},
		{Name: "Link", Type: URLProp, URL: urlJSON},
	}

	data := pa.ForUpdate()
	assert.NotNil(t, data)

	var result map[string]json.RawMessage
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Done")
	assert.Contains(t, result, "Link")
}
