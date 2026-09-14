package log

import (
	"io"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

// =============================================================================
// With field-aliasing regression (BUG-AUDIT-REPORT H4)
//
// Logger.With used to append the preset fields onto the parent's own slice:
//
//	fields: append(l.fields, toFields(args...)...)
//
// When that slice had spare capacity, two children of the same parent appended into
// the same spare slot and the later child silently overwrote the earlier child's
// field. A With chain of two or more hops always produces spare capacity, because
// append doubles the capacity it returns, so the corruption is common rather than
// exotic. It is invisible to -race when the children are built sequentially.
// =============================================================================

// capture collects the Records a capturing Sink receives.
type capture struct {
	sink    *Sink
	mu      sync.Mutex
	records []Record
}

// newCapture returns a capture whose sink accepts every level.
func newCapture() *capture {
	c := &capture{}
	c.sink = newSink(NewTextEncoder(false), io.Discard, WithLevel(TraceLevel))
	c.sink.logFunc = func(r Record) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.records = append(c.records, r)
	}
	return c
}

// all returns a copy of the records received so far.
func (c *capture) all() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Record(nil), c.records...)
}

// lookupValue returns the value logged under key and whether the key is present.
func lookupValue(fields []Field, key string) (any, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f.Value, true
		}
	}
	return nil, false
}

// loggedValue returns the value logged under key, failing the test when absent.
func loggedValue(t *testing.T, fields []Field, key string) any {
	t.Helper()
	v, ok := lookupValue(fields, key)
	if !ok {
		t.Fatalf("field %q missing from %+v", key, fields)
	}
	return v
}

// TestLoggerWith_SiblingChildrenKeepTheirOwnFields is the audit's exact scenario: a
// parent holding spare capacity, then two siblings preset with the same key. Each
// child must log its own value.
func TestLoggerWith_SiblingChildrenKeepTheirOwnFields(t *testing.T) {
	c := newCapture()
	// Spare capacity is the precondition the append-based With relied on. It is built
	// explicitly so the regression is deterministic instead of depending on the
	// capacity a particular append chain happens to grow to.
	parent := &Logger{sinks: []*Sink{c.sink}, fields: make([]Field, 1, 4)}
	parent.fields[0] = Field{Key: "shared", Value: "base"}

	x := parent.With("k", "X")
	y := parent.With("k", "Y")

	// Create both siblings before logging either: the corruption happened at
	// construction time, not at log time.
	x.Info("x")
	y.Info("y")

	recs := c.all()
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if got := loggedValue(t, recs[0].Fields, "k"); got != "X" {
		t.Errorf(`first sibling logged k=%v, want "X" (second sibling overwrote it)`, got)
	}
	if got := loggedValue(t, recs[1].Fields, "k"); got != "Y" {
		t.Errorf(`second sibling logged k=%v, want "Y"`, got)
	}
	for i, r := range recs {
		if got := loggedValue(t, r.Fields, "shared"); got != "base" {
			t.Errorf("record %d lost the inherited field: shared=%v, want %q", i, got, "base")
		}
	}
}

// TestLoggerWith_MultiHopChainIsolatedFields covers >=2-hop chains and the
// same-parent-many-children shape across several chain depths.
func TestLoggerWith_MultiHopChainIsolatedFields(t *testing.T) {
	tests := []struct {
		name         string
		parent       func(l *Logger) *Logger
		wantAncestor map[string]any
	}{
		{
			name:         "one hop",
			parent:       func(l *Logger) *Logger { return l.With("a", 1) },
			wantAncestor: map[string]any{"a": 1},
		},
		{
			name:         "two hops",
			parent:       func(l *Logger) *Logger { return l.With("a", 1).With("b", 2) },
			wantAncestor: map[string]any{"a": 1, "b": 2},
		},
		{
			name:         "three hops",
			parent:       func(l *Logger) *Logger { return l.With("a", 1).With("b", 2).With("c", 3) },
			wantAncestor: map[string]any{"a": 1, "b": 2, "c": 3},
		},
		{
			name: "four hops",
			parent: func(l *Logger) *Logger {
				return l.With("a", 1).With("b", 2).With("c", 3).With("d", 4)
			},
			wantAncestor: map[string]any{"a": 1, "b": 2, "c": 3, "d": 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCapture()
			parent := tt.parent(New(c.sink))

			first := parent.With("k", "first")
			second := parent.With("k", "second")
			first.Info("first")
			second.Info("second")

			recs := c.all()
			if len(recs) != 2 {
				t.Fatalf("got %d records, want 2", len(recs))
			}
			if got := loggedValue(t, recs[0].Fields, "k"); got != "first" {
				t.Errorf("first child logged k=%v, want %q", got, "first")
			}
			if got := loggedValue(t, recs[1].Fields, "k"); got != "second" {
				t.Errorf("second child logged k=%v, want %q", got, "second")
			}
			for i, r := range recs {
				for key, want := range tt.wantAncestor {
					if got := loggedValue(t, r.Fields, key); got != want {
						t.Errorf("record %d: %s=%v, want %v", i, key, got, want)
					}
				}
			}

			// Property: building yet another sibling must not disturb the children that
			// already exist. Pre-fix this append landed in the parent's spare slot, which
			// was the same storage first and second had just written to.
			before := append([]Field(nil), first.fields...)
			_ = parent.With("k", "third")
			if !reflect.DeepEqual(first.fields, before) {
				t.Errorf("building a third sibling mutated the first: fields = %+v, want %+v",
					first.fields, before)
			}
		})
	}
}

// TestLoggerWith_DeepChainIsolatedFields walks a four-level chain where every level
// has siblings, so spare capacity exists at each generation.
func TestLoggerWith_DeepChainIsolatedFields(t *testing.T) {
	c := newCapture()
	root := New(c.sink).With("a", 1)
	mid := root.With("b", 2)
	s1 := mid.With("k", "s1")
	s2 := mid.With("k", "s2")
	g1 := s1.With("g", "g1")
	g2 := s1.With("g", "g2")

	root.Info("root")
	mid.Info("mid")
	s1.Info("s1")
	s2.Info("s2")
	g1.Info("g1")
	g2.Info("g2")

	want := []struct {
		msg string
		k   any
		g   any
	}{
		{"root", nil, nil},
		{"mid", nil, nil},
		{"s1", "s1", nil},
		{"s2", "s2", nil},
		{"g1", "s1", "g1"},
		{"g2", "s1", "g2"},
	}
	recs := c.all()
	if len(recs) != len(want) {
		t.Fatalf("got %d records, want %d", len(recs), len(want))
	}
	for i, w := range want {
		r := recs[i]
		if r.Message != w.msg {
			t.Fatalf("record %d is %q, want %q", i, r.Message, w.msg)
		}
		// An absent key reads as nil, which is exactly what the top levels expect.
		if got, _ := lookupValue(r.Fields, "k"); got != w.k {
			t.Errorf("%s: k=%v, want %v", w.msg, got, w.k)
		}
		if got, _ := lookupValue(r.Fields, "g"); got != w.g {
			t.Errorf("%s: g=%v, want %v", w.msg, got, w.g)
		}
		if got := loggedValue(t, r.Fields, "a"); got != 1 {
			t.Errorf("%s: lost root field a=%v, want 1", w.msg, got)
		}
	}
}

// TestLoggerWith_ConcurrentSiblings builds siblings of one parent from many
// goroutines. Pre-fix they append into the same shared slot, which corrupts values
// and is a genuine data race for -race; post-fix the parent is only read.
func TestLoggerWith_ConcurrentSiblings(t *testing.T) {
	const goroutines = 32

	c := newCapture()
	// Spare capacity again, this time wide enough that every concurrent append lands
	// in the shared array.
	parent := &Logger{sinks: []*Sink{c.sink}, fields: make([]Field, 1, 64)}
	parent.fields[0] = Field{Key: "shared", Value: "base"}

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := strconv.Itoa(i)
			// The preset field and the per-call echo share a key, so each record can
			// check itself without coordinating with the other goroutines.
			parent.With("worker", id).Info("done", "worker", id)
		}(i)
	}
	wg.Wait()

	recs := c.all()
	if len(recs) != goroutines {
		t.Fatalf("got %d records, want %d", len(recs), goroutines)
	}
	seen := make(map[string]bool, goroutines)
	for _, r := range recs {
		var ids []string
		for _, f := range r.Fields {
			if f.Key != "worker" {
				continue
			}
			s, ok := f.Value.(string)
			if !ok {
				t.Fatalf("worker field is %T, want string", f.Value)
			}
			ids = append(ids, s)
		}
		if len(ids) != 2 {
			t.Fatalf("record has %d worker fields, want 2 (preset + per-call)", len(ids))
		}
		if ids[0] != ids[1] {
			t.Errorf("record corrupted: preset worker=%q but per-call worker=%q", ids[0], ids[1])
		}
		seen[ids[0]] = true
	}
	if len(seen) != goroutines {
		t.Errorf("saw %d distinct workers, want %d (fields overwrote each other)", len(seen), goroutines)
	}
}

// TestLoggerWith_EmptyAndOddArgs pins the edge-case behavior so the copy cannot
// regress it: a no-argument With copies the parent's fields, malformed arguments add
// nothing, and a zero-value Logger stays usable.
func TestLoggerWith_EmptyAndOddArgs(t *testing.T) {
	c := newCapture()
	base := New(c.sink).With("a", 1)

	tests := []struct {
		name string
		got  *Logger
		want []Field
	}{
		{
			name: "no args copies the parent fields",
			got:  base.With(),
			want: []Field{{Key: "a", Value: 1}},
		},
		{
			name: "dangling key adds nothing",
			got:  base.With("lonely"),
			want: []Field{{Key: "a", Value: 1}},
		},
		{
			name: "non-string key adds nothing",
			got:  base.With(42, "x"),
			want: []Field{{Key: "a", Value: 1}},
		},
		{
			name: "zero-value logger with no fields stays empty",
			got:  (&Logger{}).With(),
			want: nil,
		},
		{
			name: "zero-value logger keeps the fields it is given",
			got:  (&Logger{}).With("a", 1),
			want: []Field{{Key: "a", Value: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got.fields, tt.want) {
				t.Errorf("fields = %+v, want %+v", tt.got.fields, tt.want)
			}
		})
	}

	// A no-argument child must still be isolated: writing to its own descendants must
	// not disturb it or its siblings.
	child := base.With()
	sibling := base.With("k", "sibling")
	_ = child.With("k", "grandchild")
	if got := loggedValue(t, child.fields, "a"); got != 1 {
		t.Errorf("child lost its parent field: a=%v, want 1", got)
	}
	if got := loggedValue(t, sibling.fields, "k"); got != "sibling" {
		t.Errorf("sibling field corrupted: k=%v, want %q", got, "sibling")
	}
	if got := loggedValue(t, base.fields, "a"); got != 1 {
		t.Errorf("parent field corrupted: a=%v, want 1", got)
	}
}
