package alfred

import (
	"bytes"
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
