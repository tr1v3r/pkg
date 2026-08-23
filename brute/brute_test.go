package brute

import (
	"errors"
	"testing"
)

// mazeState implements State for a 2D grid maze search.
type mazeState struct {
	x, y    int
	maze    []string
	failPre bool
}

func (m mazeState) Key() string { return string(rune('0'+m.x)) + "," + string(rune('0'+m.y)) }

func (m mazeState) Preprocess() error {
	if m.failPre {
		return errors.New("preprocess failed")
	}
	return nil
}

func (m mazeState) Done() bool {
	return m.maze[m.y][m.x] == 'E'
}

func (m mazeState) next() []mazeState {
	var states []mazeState
	dirs := [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} // up, right, down, left
	for _, d := range dirs {
		nx, ny := m.x+d[0], m.y+d[1]
		if ny < 0 || ny >= len(m.maze) || nx < 0 || nx >= len(m.maze[ny]) {
			continue
		}
		if m.maze[ny][nx] == '#' {
			continue
		}
		states = append(states, mazeState{x: nx, y: ny, maze: m.maze})
	}
	return states
}

var testMaze = []string{
	"S.#.",
	".#..",
	".#.#",
	"...E",
}

func TestBruter_FindBFS(t *testing.T) {
	b := NewBruter[mazeState](func(s mazeState) []mazeState { return s.next() })

	final, err := b.Find(mazeState{x: 0, y: 0, maze: testMaze}, BFS)
	if err != nil {
		t.Fatalf("BFS find fail: %s", err)
	}
	if final == nil {
		t.Fatal("BFS find fail: got nil final step")
	}

	steps := final.Backtrack()
	// S(0,0) -> E(3,3): shortest path is 7 cells (down the left column, then right)
	if len(steps) != 7 {
		t.Errorf("BFS should find shortest path of 7 steps, got %d", len(steps))
	}
	if !final.State.Done() {
		t.Error("final state should be Done")
	}
	if got := final.Cost(); got != 6 {
		t.Errorf("final cost = %d, want 6", got)
	}
	if steps[0].parent != nil {
		t.Error("first step of backtrack should have no parent")
	}
	if steps[len(steps)-1] != final {
		t.Error("last backtrack step should be the final step")
	}
}

func TestBruter_FindDFS(t *testing.T) {
	b := NewBruter[mazeState](func(s mazeState) []mazeState { return s.next() })

	final, err := b.Find(mazeState{x: 0, y: 0, maze: testMaze}, DFS)
	if err != nil {
		t.Fatalf("DFS find fail: %s", err)
	}
	if final == nil {
		t.Fatal("DFS find fail: got nil final step")
	}

	steps := final.Backtrack()
	if len(steps) < 6 {
		t.Errorf("DFS path should be at least shortest length, got %d", len(steps))
	}
	if !final.State.Done() {
		t.Error("final state should be Done")
	}
	if final.Cost() != len(steps)-1 {
		t.Errorf("cost %d should equal len(steps)-1 = %d", final.Cost(), len(steps)-1)
	}
}

func TestBruter_FindUnknownMethod(t *testing.T) {
	b := NewBruter[mazeState](func(s mazeState) []mazeState { return s.next() })

	final, err := b.Find(mazeState{x: 0, y: 0, maze: testMaze}, METHOD("unknown"))
	if final != nil {
		t.Errorf("unknown method should return nil step, got %v", final)
	}
	if err == nil {
		t.Error("unknown method should return error")
	}
}

func TestBruter_FindPreprocessError(t *testing.T) {
	b := NewBruter[mazeState](func(s mazeState) []mazeState { return s.next() })

	start := mazeState{x: 0, y: 0, maze: testMaze, failPre: true}
	final, err := b.Find(start, BFS)
	if final != nil {
		t.Errorf("preprocess error should return nil step, got %v", final)
	}
	if err == nil {
		t.Error("preprocess error should return error")
	}
}

func TestBruter_FindNoSolution(t *testing.T) {
	blocked := []string{
		"S.#E",
		"###.",
	}
	b := NewBruter[mazeState](func(s mazeState) []mazeState { return s.next() })

	for _, method := range []METHOD{BFS, DFS} {
		final, err := b.Find(mazeState{x: 0, y: 0, maze: blocked}, method)
		if err != nil {
			t.Fatalf("%v find fail: %s", method, err)
		}
		if final != nil {
			t.Errorf("%v should not find a path in blocked maze", method)
		}
	}
}

func TestBruter_FindReachAdjacentDone(t *testing.T) {
	maze := []string{"E."}
	b := NewBruter[mazeState](func(s mazeState) []mazeState { return s.next() })

	// done-ness is only detected on successor states, so search from the open
	// cell (1,0) whose neighbor (0,0) is the goal.
	final, err := b.Find(mazeState{x: 1, y: 0, maze: maze}, BFS)
	if err != nil {
		t.Fatalf("find fail: %s", err)
	}
	if final == nil {
		t.Fatal("should find the adjacent done state")
	}
	if steps := final.Backtrack(); len(steps) != 2 {
		t.Errorf("expected 2 steps to reach E, got %d", len(steps))
	}

	// a start state that is already Done is not detected by the search loop
	final, err = b.Find(mazeState{x: 0, y: 0, maze: maze}, BFS)
	if err != nil {
		t.Fatalf("find fail: %s", err)
	}
	if final != nil {
		t.Errorf("done start state is not detected by the loop, want nil, got %v", final)
	}
}

func TestStep_BacktrackNil(t *testing.T) {
	var s *Step[mazeState]
	if steps := s.Backtrack(); steps != nil {
		t.Errorf("nil step backtrack should return nil, got %v", steps)
	}
}

func TestStep_NewStepChain(t *testing.T) {
	first := NewStep[mazeState](mazeState{x: 1, y: 1, maze: testMaze}, nil)
	second := NewStep[mazeState](mazeState{x: 2, y: 1, maze: testMaze}, first)
	third := NewStep[mazeState](mazeState{x: 3, y: 1, maze: testMaze}, second)

	if first.Cost() != 0 {
		t.Errorf("root cost = %d, want 0", first.Cost())
	}
	if second.Cost() != 1 {
		t.Errorf("second cost = %d, want 1", second.Cost())
	}
	if third.Cost() != 2 {
		t.Errorf("third cost = %d, want 2", third.Cost())
	}

	steps := third.Backtrack()
	if len(steps) != 3 {
		t.Fatalf("backtrack len = %d, want 3", len(steps))
	}
	if steps[0] != first || steps[1] != second || steps[2] != third {
		t.Error("backtrack order wrong")
	}
}

func TestStep_Visited(t *testing.T) {
	first := NewStep[mazeState](mazeState{x: 1, y: 1, maze: testMaze}, nil)
	second := NewStep[mazeState](mazeState{x: 2, y: 1, maze: testMaze}, first)

	if !second.visited(mazeState{x: 1, y: 1}.Key()) {
		t.Error("parent key should be visited")
	}
	if !second.visited(mazeState{x: 2, y: 1}.Key()) {
		t.Error("own key should be visited")
	}
	if second.visited(mazeState{x: 3, y: 1}.Key()) {
		t.Error("unrelated key should not be visited")
	}
}

func TestQueue(t *testing.T) {
	var q Queue[mazeState]

	if !q.Empty() {
		t.Error("new queue should be empty")
	}
	if s := q.Dequeue(); s != nil {
		t.Errorf("dequeue on empty queue should return nil, got %v", s)
	}

	s1 := NewStep[mazeState](mazeState{x: 1, y: 1, maze: testMaze}, nil)
	s2 := NewStep[mazeState](mazeState{x: 2, y: 1, maze: testMaze}, nil)
	s3 := NewStep[mazeState](mazeState{x: 3, y: 1, maze: testMaze}, nil)

	q.Enqueue(s1)
	q.Enqueue(s2, s3)

	if q.Empty() {
		t.Error("queue with items should not be empty")
	}
	if got := q.Dequeue(); got != s1 {
		t.Errorf("dequeue should be FIFO, want first item")
	}
	if got := q.Dequeue(); got != s2 {
		t.Errorf("dequeue should be FIFO, want second item")
	}
	if got := q.Dequeue(); got != s3 {
		t.Errorf("dequeue should be FIFO, want third item")
	}
	if !q.Empty() {
		t.Error("queue should be empty after draining")
	}
}
