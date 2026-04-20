package notion

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCondition_QueryParams(t *testing.T) {
	cond := &Condition{
		FilterProperties: []string{"title", "status"},
	}

	params := cond.QueryParams()
	assert.Contains(t, params, "filter_properties")
	assert.Contains(t, params, "title")
	assert.Contains(t, params, "status")
}

func TestCondition_Payload_Full(t *testing.T) {
	cond := &Condition{
		PageSize:    10,
		StartCursor: "cursor-abc",
		Filter: &FilterCondition{
			FilterSingleCondition: FilterSingleCondition{
				Property: "Status",
				Status:   &StatusFilter{Equals: "Done"},
			},
		},
		Sorts: []PropSortCondition{
			{Property: "Name", Direction: "ascending"},
		},
	}

	data := cond.Payload()
	assert.NotNil(t, data)

	var result map[string]any
	err := json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, float64(10), result["page_size"])
	assert.Equal(t, "cursor-abc", result["start_cursor"])
	assert.NotNil(t, result["filter"])
	assert.NotNil(t, result["sorts"])
}

func TestCondition_Payload_Empty(t *testing.T) {
	cond := &Condition{}

	data := cond.Payload()
	assert.Nil(t, data)
}

func TestCondition_Payload_Nil(t *testing.T) {
	var cond *Condition
	data := cond.Payload()
	assert.Nil(t, data)
}

func TestFilterCondition_SingleFilter(t *testing.T) {
	cond := &FilterCondition{
		FilterSingleCondition: FilterSingleCondition{
			Property: "Priority",
			Select:   &SelectFilter{Equals: "High"},
		},
	}

	data, err := json.Marshal(cond)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "Priority", result["property"])

	sel, ok := result["select"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "High", sel["equals"])
}

func TestFilterCondition_CompoundFilter_And(t *testing.T) {
	cond := &FilterCondition{
		CompoundConditions: map[string][]FilterCondition{
			"and": {
				{
					FilterSingleCondition: FilterSingleCondition{
						Property: "Done",
						CheckBox: &CheckBoxFilter{Equals: true},
					},
				},
				{
					FilterSingleCondition: FilterSingleCondition{
						Property: "Priority",
						Select:   &SelectFilter{Equals: "High"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(cond)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, "and")

	andFilters, ok := result["and"].([]any)
	assert.True(t, ok)
	assert.Len(t, andFilters, 2)
}

func TestFilterCondition_CompoundFilter_Or(t *testing.T) {
	cond := &FilterCondition{
		CompoundConditions: map[string][]FilterCondition{
			"or": {
				{
					FilterSingleCondition: FilterSingleCondition{
						Property: "Tags",
						Select:   &SelectFilter{Equals: "Go"},
					},
				},
				{
					FilterSingleCondition: FilterSingleCondition{
						Property: "Tags",
						Select:   &SelectFilter{Equals: "Python"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(cond)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Contains(t, result, "or")

	orFilters, ok := result["or"].([]any)
	assert.True(t, ok)
	assert.Len(t, orFilters, 2)
}

func TestSelectFilter(t *testing.T) {
	equals := &SelectFilter{Equals: "Active"}
	data, err := json.Marshal(equals)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "Active", result["equals"])

	doesNotEqual := &SelectFilter{DoesNotEqual: "Archived"}
	data, err = json.Marshal(doesNotEqual)
	assert.NoError(t, err)

	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "Archived", result["does_not_equal"])
}

func TestDateFilter(t *testing.T) {
	before := &DateFilter{Before: "2024-12-31"}
	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "2024-12-31", result["before"])

	after := &DateFilter{After: "2024-01-01"}
	data, err = json.Marshal(after)
	assert.NoError(t, err)

	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "2024-01-01", result["after"])
}

func TestRichTextFilter(t *testing.T) {
	contains := &RichTextFilter{Contains: "hello"}
	data, err := json.Marshal(contains)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "hello", result["contains"])

	equals := &RichTextFilter{Equals: "exact match"}
	data, err = json.Marshal(equals)
	assert.NoError(t, err)

	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "exact match", result["equals"])
}

func TestNumberFilter(t *testing.T) {
	val := 100.0
	greaterThan := &NumberFilter{GreaterThan: &val}
	data, err := json.Marshal(greaterThan)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, val, result["greater_than"])

	equalsVal := 42.0
	equals := &NumberFilter{Equals: &equalsVal}
	data, err = json.Marshal(equals)
	assert.NoError(t, err)

	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, equalsVal, result["equals"])
}

func TestStatusFilter(t *testing.T) {
	status := &StatusFilter{Equals: "In Progress"}
	data, err := json.Marshal(status)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "In Progress", result["equals"])
}

func TestPropSortCondition(t *testing.T) {
	sort := PropSortCondition{
		Property:  "Created Time",
		Direction: "descending",
	}

	data, err := json.Marshal(sort)
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, "Created Time", result["property"])
	assert.Equal(t, "descending", result["direction"])
}
