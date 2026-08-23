package alfred

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestOutput(t *testing.T) {
	wf := NewWorkFlow(&FlowItem{Title: "new item"})

	expect := []byte(`{"items":[{"title":"new item","subtitle":"","arg":""}]}`)
	data := wf.Output()
	if !bytes.Equal(expect, data) {
		t.Errorf("output error, expect %s, got %s", expect, data)
	}
}

func TestNewWorkFlow_MultipleItems(t *testing.T) {
	wf := NewWorkFlow(
		&FlowItem{Title: "one", Arg: "1"},
		&FlowItem{Title: "two", Arg: "2"},
	)
	if len(wf.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(wf.Items))
	}

	var out struct {
		Items []FlowItem `json:"items"`
	}
	if err := json.Unmarshal(wf.Output(), &out); err != nil {
		t.Fatalf("unmarshal output fail: %s", err)
	}
	if len(out.Items) != 2 || out.Items[0].Title != "one" || out.Items[1].Title != "two" {
		t.Errorf("output items = %+v", out.Items)
	}
}

func TestWorkFlow_FullFields(t *testing.T) {
	valid := false
	wf := NewWorkFlow(&FlowItem{Title: "x"})
	wf.Vars = map[string]interface{}{"k": "v"}
	wf.Rerun = 1.5
	wf.SkipKnowledge = true
	wf.Items[0].Valid = &valid
	wf.Items[0].Match = "match field"
	wf.Items[0].AutoComplete = "auto"
	wf.Items[0].Type = "file"
	wf.Items[0].Mods = map[ModifierKey]ItemReact{
		AltKey: {Valid: true, Arg: "alt-arg", Subtitle: "alt"},
	}
	wf.Items[0].Action = []string{"https://example.com"}
	wf.Items[0].Text = &ItemTextReact{Copy: "copied", LargetType: "large"}
	wf.Items[0].QuickLookURL = "https://example.com/quick"
	wf.Items[0].Icon = &ItemIcon{Type: "fileicon", Path: "~/Desktop"}

	var out map[string]interface{}
	if err := json.Unmarshal(wf.Output(), &out); err != nil {
		t.Fatalf("unmarshal output fail: %s", err)
	}

	if vars, _ := out["variables"].(map[string]interface{}); vars["k"] != "v" {
		t.Errorf("variables = %v", out["variables"])
	}
	if out["rerun"] != 1.5 {
		t.Errorf("rerun = %v", out["rerun"])
	}
	if out["skipknowledge"] != true {
		t.Errorf("skipknowledge = %v", out["skipknowledge"])
	}

	items := out["items"].([]interface{})
	item := items[0].(map[string]interface{})
	for _, key := range []string{"valid", "match", "autocomplete", "type", "mods", "action", "text", "quicklookurl", "icon"} {
		if _, ok := item[key]; !ok {
			t.Errorf("output missing field %q", key)
		}
	}
}

func TestWorkFlow_AddAndReset(t *testing.T) {
	wf := NewWorkFlow(&FlowItem{Title: "first"})
	wf.Vars = map[string]interface{}{"a": "b"}
	wf.Rerun = 2
	wf.SkipKnowledge = true

	wf.Add(&FlowItem{Title: "second"}, &FlowItem{Title: "third"})
	if len(wf.Items) != 3 {
		t.Fatalf("after Add items = %d, want 3", len(wf.Items))
	}

	wf.Reset()
	if len(wf.Items) != 0 {
		t.Errorf("after Reset items = %d, want 0", len(wf.Items))
	}
	if wf.Vars != nil {
		t.Errorf("after Reset vars = %v, want nil", wf.Vars)
	}
	if wf.Rerun != 0 {
		t.Errorf("after Reset rerun = %v, want 0", wf.Rerun)
	}
	if wf.SkipKnowledge {
		t.Error("after Reset skipknowledge should be false")
	}
}

func TestWorkFlow_WriteTo(t *testing.T) {
	wf := NewWorkFlow(&FlowItem{Title: "wt", Arg: "a1"})

	var buf bytes.Buffer
	n, err := wf.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo fail: %s", err)
	}
	if int(n) != buf.Len() {
		t.Errorf("WriteTo n = %d, buffer len = %d", n, buf.Len())
	}
	if !strings.Contains(buf.String(), "wt") {
		t.Errorf("buffer = %q, want to contain item title", buf.String())
	}
}

func TestWorkFlow_Print(t *testing.T) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe fail: %s", err)
	}
	os.Stdout = w

	wf := NewWorkFlow(&FlowItem{Title: "printed"})
	n, err := wf.Print()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Print fail: %s", err)
	}
	if n == 0 {
		t.Error("Print wrote 0 bytes")
	}
	if !strings.Contains(buf.String(), "printed") {
		t.Errorf("stdout = %q, want to contain item title", buf.String())
	}
}

func TestFlowItem_Duplicate(t *testing.T) {
	original := &FlowItem{Title: "dup", Arg: "arg", UID: "uid"}
	cp := original.Duplicate()

	if cp == original {
		t.Fatal("Duplicate should return a copy, not the same pointer")
	}
	if cp.Title != original.Title || cp.Arg != original.Arg || cp.UID != original.UID {
		t.Errorf("duplicate = %+v, want %+v", cp, original)
	}

	// mutating the copy must not affect the original
	cp.Title = "changed"
	if original.Title != "dup" {
		t.Error("mutating duplicate affected the original")
	}
}

func TestModifierKey_Combine(t *testing.T) {
	if got := AltKey.Combine(CtrlKey, CmdKey); string(got) != "alt+ctrl+cmd" {
		t.Errorf("Alt+Ctrl+Cmd = %q, want %q", got, "alt+ctrl+cmd")
	}
	if got := ShiftKey.Combine(FnKey); string(got) != "shift+fn" {
		t.Errorf("Shift+Fn = %q, want %q", got, "shift+fn")
	}
	if got := CmdKey.Combine(); string(got) != "cmd" {
		t.Errorf("Cmd alone = %q, want %q", got, "cmd")
	}
}
