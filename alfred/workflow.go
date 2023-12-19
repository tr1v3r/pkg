package alfred

import (
	"encoding/json"
	"io"
	"os"
)

// 官方说明文档：https://www.alfredapp.com/help/workflows/inputs/script-filter/json/

// NewWorkFlow create new alfred workflow
func NewWorkFlow(items ...*FlowItem) *WorkFlow {
	return &WorkFlow{Items: items}
}

// WorkFlow 一个工作流实体
type WorkFlow struct {
	// Vars variables within a variables object will be passed out of the script filter and remain accessible throughout the current session as environment variables.
	Vars map[string]interface{} `json:"variables,omitempty"` // variables
	// Rerun scripts can be set to re-run automatically after an interval using the rerun key with a value from 0.1 to 5.0 seconds. The script will only be re-run if the script filter is still active and the user hasn't changed the state of the filter by typing and triggering a re-run.
	Rerun float64 `json:"rerun,omitempty"`
	// SkipKnowledge do not use alfred auto sort order
	SkipKnowledge bool `json:"skipknowledge,omitempty"`
	// Items each item describes a result row displayed in Alfred. The three obvious elements are the ones you see in an Alfred result row - title, subtitle and icon.
	Items []*FlowItem `json:"items"`
}

// AddItem 增加显示条目
func (wf *WorkFlow) Add(items ...*FlowItem) {
	wf.Items = append(wf.Items, items...)
}

// Output 输出items
func (wf *WorkFlow) Output() []byte {
	data, _ := json.Marshal(wf)
	return data
}

// WriteTo output to writer
func (wf *WorkFlow) WriteTo(w io.Writer) (n int64, err error) {
	m, err := w.Write(wf.Output())
	return int64(m), err
}

// Print equal to .WriteTo(os.Stdout)
func (wf *WorkFlow) Print() (n int64, err error) {
	return wf.WriteTo(os.Stdout)
}

// Reset 重置
func (wf *WorkFlow) Reset() {
	wf.Vars = nil
	wf.Rerun = 0
	wf.Items = nil
}

// FlowItem 一条工作信息
type FlowItem struct {
	// UID (optional) a unique identifier for the item. It allows Alfred to learn about the item for subsequent sorting and ordering of the user's actioned results.
	UID string `json:"uid,omitempty"`
	// Title (required) the title displayed in the result row. There are no options for this element and it is essential that this element is populated.
	Title string `json:"title"`
	// Subtitle (optional) The subtitle displayed in the result row.
	Subtitle string `json:"subtitle"`
	// Arg (string/array recommended)) the argument which is passed through the workflow to the connected output action.
	Arg string `json:"arg"`
	// Icon (optional) the icon displayed in the result row. path is relative to the workflow's root folder
	Icon *ItemIcon `json:"icon,omitempty"`
	// Valid (optional, default=true)
	Valid *bool `json:"valid,omitempty"`
	// Match (optional) the match field enables you to define what Alfred matches against when the workflow is set to "Alfred Filters Results". If match is present, it fully replaces matching on the title property.
	Match string `json:"match,omitempty"`
	// AutoComplete (recommended)
	AutoComplete string `json:"autocomplete,omitempty"`
	// Type ("default" | "file" | "file:skipcheck" optional, default = "default")
	// by specifying "type": "file", Alfred treats your result as a file on your system. This allows the user to perform actions on the file like they can with Alfred's standard file filters.
	Type string `json:"type,omitempty"`
	// Mods (optional) the mod element gives you control over how the modifier keys react. It can alter the looks of a result (e.g. subtitle, icon) and output a different arg or session variables.
	Mods map[ModifierKey]ItemReact `json:"mods,omitempty"`
	// Action (OBJECT | ARRAY | STRING optional) this element defines the Universal Action items used when actioning the result, and overrides the arg being used for actioning. The action key can take a string or array for simple types, and the content type will automatically be derived by Alfred to file, url, or text.
	Action any `json:"action"`
	// Text (optional) Defines the text the user will get when copying the selected result row with ⌘C or displaying large type with ⌘L.
	Text *ItemTextReact `json:"text,omitempty"`
	// QuickLookURL (optional) a Quick Look URL which will be visible if the user uses the Quick Look feature within Alfred (tapping shift, or ⌘Y). quicklookurl will also accept a file path, both absolute and relative to home using ~/.
	QuickLookURL string `json:"quicklookurl,omitempty"`
}

// Duplicate duplicate flow item
func (item FlowItem) Duplicate() *FlowItem { return &item }

// ItemIcon item的图标
type ItemIcon struct {
	Type string `json:"type,omitempty"`
	Path string `json:"path,omitempty"`
}

// ItemReact 不同按键的触发反应
type ItemReact struct {
	Valid    bool   `json:"valid"`
	Arg      string `json:"arg"`
	Subtitle string `json:"subtitle"`
}

// ItemTextReact 文本操作
type ItemTextReact struct {
	Copy       string `json:"copy,omitempty"`
	LargetType string `json:"largetype,omitempty"`
}

// ModifierKey modifier key
type ModifierKey string

const (
	AltKey   ModifierKey = "alt"
	CmdKey   ModifierKey = "cmd"
	CtrlKey  ModifierKey = "ctrl"
	ShiftKey ModifierKey = "shift"
	FnKey    ModifierKey = "fn"
)

func (k ModifierKey) Combine(keys ...ModifierKey) ModifierKey {
	for _, key := range keys {
		k += "+" + key
	}
	return k
}
