package notion

import (
	"encoding/json"
)

// Condition query filter
type Condition struct {
	PageSize    int                 `json:"page_size,omitempty"`
	StartCursor string              `json:"start_cursor,omitempty"`
	Filter      *FilterCondition    `json:"filter,omitempty"`
	Sorts       []PropSortCondition `json:"sorts,omitempty"`
}

// Payload return payload
func (f *Condition) Payload() (data []byte) {
	if f == nil {
		return nil
	}

	var payload = make(map[string]interface{}, 4)

	if f.PageSize != 0 {
		payload["page_size"] = f.PageSize
	}
	if f.StartCursor != "" {
		payload["start_cursor"] = f.StartCursor
	}
	if f.Filter != nil {
		payload["filter"] = f.Filter
	}
	if f.Sorts != nil {
		payload["sorts"] = f.Sorts
	}

	if len(payload) == 0 {
		return nil
	}

	data, _ = json.Marshal(payload)
	return data
}

// FilterCondition ...
//
//	{
//	  "and": [
//	    {
//	      "property": "Done",
//	      "checkbox": {
//	        "equals": true
//	      }
//	    },
//	    {
//	      "or": [
//	        {
//	          "property": "Tags",
//	          "contains": "A"
//	        },
//	        {
//	          "property": "Tags",
//	          "contains": "B"
//	        }
//	      ]
//	    }
//	  ]
//	}
type FilterCondition struct {
	FilterSingleCondition

	CompoundConditions map[string][]FilterCondition
}

func (cond *FilterCondition) MarshalJSON() ([]byte, error) {
	if cond.CompoundConditions != nil {
		return json.Marshal(cond.CompoundConditions)
	}
	return json.Marshal(cond.FilterSingleCondition)
}

// FilterSingleCondition filter single condition
// https://developers.notion.com/reference/post-database-query-filter#the-filter-object
type FilterSingleCondition struct {
	Property string          `json:"property"`
	CheckBox *CheckBoxFilter `json:"checkbox,omitempty"`
	RichText *RichTextFilter `json:"rich_text,omitempty"`
	Number   *NumberFilter   `json:"number,omitempty"`
	Files    *FilesFilter    `json:"files,omitempty"`

	Contains string `json:"contains,omitempty"`
}

// CheckBoxFilter ...
type CheckBoxFilter struct {
	Equals       bool `json:"equals"`
	DoesNotEqual bool `json:"does_not_equal"`
}

// RichTextFilter ...
type RichTextFilter struct {
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	Equals         string `json:"equals,omitempty"`
	DoesNotEqual   string `json:"does_not_equal,omitempty"`
	StartsWith     string `json:"starts_with,omitempty"`
	EndsWith       string `json:"ends_with,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// NumberFilter ...
// doc: https://developers.notion.com/reference/post-database-query-filter#number
type NumberFilter struct {
	Equals               float64 `json:"equals,omitempty"`
	DoesNotEqual         float64 `json:"does_not_equal,omitempty"`
	GreaterThan          float64 `json:"greater_than,omitempty"`
	GreaterThanOrEqualTo float64 `json:"greater_than_or_equal_to,omitempty"`
	LessThan             float64 `json:"less_than,omitempty"`
	LessThanOrEqualTo    float64 `json:"less_than_or_equal_to,omitempty"`
	IsEmpty              bool    `json:"is_empty,omitempty"`
	IsNotEmpty           bool    `json:"is_not_empty,omitempty"`
}

// FilesFilter ...
// doc: https://developers.notion.com/reference/post-database-query-filter#files
type FilesFilter struct {
	IsEmpty    bool `json:"is_empty,omitempty"`
	IsNotEmpty bool `json:"is_not_empty,omitempty"`
}

type PropSortCondition struct {
	Property  string `json:"property"`
	Direction string `json:"direction"` // "ascending" or "descending"
}
