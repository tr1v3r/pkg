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
	Property    string             `json:"property"`
	CheckBox    *CheckBoxFilter    `json:"checkbox,omitempty"`
	Date        *DateFilter        `json:"date,omitempty"`
	Files       *FilesFilter       `json:"files,omitempty"`
	Formula     *FormulaFilter     `json:"formula,omitempty"`
	MultiSelect *MultiSelectFilter `json:"multi_select,omitempty"`
	Number      *NumberFilter      `json:"number,omitempty"`
	People      *PeopleFilter      `json:"people,omitempty"`
	Relation    *RelationFilter    `json:"relation,omitempty"`
	RichText    *RichTextFilter    `json:"rich_text,omitempty"`
	Rollup      *RollupFilter      `json:"rollup,omitempty"`
	Select      *SelectFilter      `json:"select,omitempty"`
	Status      *StatusFilter      `json:"status,omitempty"`
	Timestamp   *TimestampFilter   `json:"timestamp,omitempty"`
	ID          *IDFilter          `json:"ID,omitempty"`

	Contains string `json:"contains,omitempty"`
}

// SelectFilter ...
type SelectFilter struct {
	Equals       string `json:"equals,omitempty"`
	DoesNotEqual string `json:"does_not_equal,omitempty"`
	IsEmpty      bool   `json:"is_empty,omitempty"`
	IsNotEmpty   bool   `json:"is_not_empty,omitempty"`
}

type MultiSelectFilter struct {
	Contains       string `json:"contains,omitempty"`
	DoesNotContain string `json:"does_not_contain,omitempty"`
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// DateFilter ...
// https://developers.notion.com/reference/post-database-query-filter#date
type DateFilter struct {
	After      string    `json:"after,omitempty"`  // ISO_8601 Date
	Before     string    `json:"before,omitempty"` // ISO_8601 Date
	Equals     string    `json:"equals,omitempty"` // ISO_8601 Date
	OnOrAfter  *struct{} `json:"on_or_after,omitempty"`
	OnOrBefore *struct{} `json:"on_or_before,omitempty"`
	PastMonth  *struct{} `json:"past_month,omitempty"`
	PastWeek   *struct{} `json:"past_week,omitempty"`
	PastYear   *struct{} `json:"past_year,omitempty"`
	ThisWeek   *struct{} `json:"this_week,omitempty"`
	IsEmpty    bool      `json:"is_empty,omitempty"`
	IsNotEmpty bool      `json:"is_not_empty,omitempty"`
}

// CheckBoxFilter ...
type CheckBoxFilter struct {
	Equals       bool `json:"equals,omitempty"`
	DoesNotEqual bool `json:"does_not_equal,omitempty"`
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

// PeopleFilter ...
type PeopleFilter struct {
	Contains       string `json:"contains,omitempty"`         // UUIDv4
	DoesNotContain string `json:"does_not_contain,omitempty"` // UUIDv4
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// RelationFilter ...
type RelationFilter struct {
	Contains       string `json:"contains,omitempty"`         // UUIDv4
	DoesNotContain string `json:"does_not_contain,omitempty"` // UUIDv4
	IsEmpty        bool   `json:"is_empty,omitempty"`
	IsNotEmpty     bool   `json:"is_not_empty,omitempty"`
}

// FilesFilter ...
// doc: https://developers.notion.com/reference/post-database-query-filter#files
type FilesFilter struct {
	IsEmpty    bool `json:"is_empty,omitempty"`
	IsNotEmpty bool `json:"is_not_empty,omitempty"`
}

// FormulaFilter ...
type FormulaFilter struct {
	CheckBox *CheckBoxFilter `json:"checkbox,omitempty"`
	Date     *DateFilter     `json:"date,omitempty"`
	Number   *NumberFilter   `json:"number,omitempty"`
	RichText *RichTextFilter `json:"rich_text,omitempty"`
}

// RollupFilter ...
// https://developers.notion.com/reference/post-database-query-filter#rollup
type RollupFilter struct {
	Any   map[string]any `json:"any,omitempty"`
	Every map[string]any `json:"every,omitempty"`
	None  map[string]any `json:"none,omitempty"`
}

// StatusFilter ...
type StatusFilter struct {
	Equals       string `json:"equals,omitempty"`
	DoesNotEqual string `json:"does_not_equal,omitempty"`
	IsEmpty      bool   `json:"is_empty,omitempty"`
	IsNotEmpty   bool   `json:"is_not_empty,omitempty"`
}

// TimestampFilter ...
type TimestampFilter struct {
	Timestamp      string      `json:"timestamp,omitempty"`
	CreatedTime    *DateFilter `json:"created_time,omitempty"`
	LastEditedTime *DateFilter `json:"last_edited_time,omitempty"`
}

// IDFilter ...
// https://developers.notion.com/reference/post-database-query-filter#id
type IDFilter map[any][]NumberFilter

type PropSortCondition struct {
	Property  string `json:"property"`
	Direction string `json:"direction"` // "ascending" or "descending"
}
