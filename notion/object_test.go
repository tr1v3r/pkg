package notion

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelationObject_IDsAndJSON(t *testing.T) {
	rel := RelationObject{{ID: "r1"}, {ID: "r2"}}

	assert.Equal(t, []string{"r1", "r2"}, rel.IDs())
	assert.Empty(t, RelationObject(nil).IDs())

	data := rel.JSON()
	assert.NotEmpty(t, data)

	var back RelationObject
	assert.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, rel, back)
}

func TestRollupObject_PlainStrings(t *testing.T) {
	rollup := RollupObject{
		Type: "array",
		Array: []struct {
			Type     string       `json:"type"`
			RichText []TextObject `json:"rich_text,omitempty"`
		}{
			{Type: "rich_text", RichText: []TextObject{{PlainText: "a"}, {PlainText: "b"}}},
			{Type: "number"}, // skipped: not rich_text
			{Type: "rich_text", RichText: []TextObject{{PlainText: "c"}}},
		},
	}
	assert.Equal(t, []string{"a", "b", "c"}, rollup.PlainStrings())

	// non-array type → nil
	assert.Nil(t, RollupObject{Type: "number"}.PlainStrings())

	// empty array → nil
	assert.Nil(t, RollupObject{Type: "array"}.PlainStrings())
}

func TestProperty_GetRelationIDs_AllBranches(t *testing.T) {
	// nil relation
	assert.Nil(t, Property{}.GetRelationIDs())

	// invalid JSON
	assert.Nil(t, Property{Relation: json.RawMessage("not-json")}.GetRelationIDs())

	// valid
	p := Property{Relation: RelationObject{{ID: "x1"}, {ID: "x2"}}.JSON()}
	assert.Equal(t, []string{"x1", "x2"}, p.GetRelationIDs())
}

func TestProperty_PlainText_Branches(t *testing.T) {
	// rich text: valid, nil, invalid
	rt := TextObjectArray{{PlainText: "hello "}, {PlainText: "world"}}.JSON()
	assert.Equal(t, "hello world", Property{Type: RichTextProp, RichText: rt}.PlainText())
	assert.Equal(t, "", Property{Type: RichTextProp}.PlainText())
	assert.Equal(t, "", Property{Type: RichTextProp, RichText: json.RawMessage("bad")}.PlainText())

	// title: valid, nil, invalid
	tt := TextObjectArray{{PlainText: "T"}}.JSON()
	assert.Equal(t, "T", Property{Type: TitleProp, Title: tt}.PlainText())
	assert.Equal(t, "", Property{Type: TitleProp}.PlainText())
	assert.Equal(t, "", Property{Type: TitleProp, Title: json.RawMessage("bad")}.PlainText())

	// select: valid, nil, invalid
	sel, _ := json.Marshal(SelectOptionObject{Name: "todo"})
	assert.Equal(t, "todo", Property{Type: SelectProp, Select: sel}.PlainText())
	assert.Equal(t, "", Property{Type: SelectProp}.PlainText())
	assert.Equal(t, "", Property{Type: SelectProp, Select: json.RawMessage("bad")}.PlainText())

	// url: valid, nil, invalid
	u, _ := json.Marshal("https://example.com")
	assert.Equal(t, "https://example.com", Property{Type: URLProp, URL: u}.PlainText())
	assert.Equal(t, "", Property{Type: URLProp}.PlainText())
	assert.Equal(t, "", Property{Type: URLProp, URL: json.RawMessage("bad")}.PlainText())

	// other types → empty
	assert.Equal(t, "", Property{Type: NumberProp}.PlainText())
}

func TestFileItemArray_JSON(t *testing.T) {
	files := FileItemArray{{Name: "f1"}}
	data := files.JSON()
	assert.NotEmpty(t, data)

	var back FileItemArray
	assert.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, "f1", back[0].Name)
}
