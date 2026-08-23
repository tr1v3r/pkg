package brute

import (
	"errors"
	"slices"
)

// METHOD selects the search strategy for Bruter.Find.
type METHOD string

const (
	// BFS is the breadth-first search method.
	BFS METHOD = "bfs"
	// DFS is the depth-first search method.
	DFS METHOD = "dfs"
)

// State is a circuit breaker state constant.
type State interface {
	Key() string
	Preprocess() error
	Done() bool
}

// NewStep creates a new step.
func NewStep[S State](s S, lastStep *Step[S]) *Step[S] {
	if lastStep == nil {
		return &Step[S]{State: s}
	}
	return &Step[S]{
		State: s,

		cost:   lastStep.cost + 1,
		parent: lastStep,
	}
}

// Step is one node in the search tree, linked to its parent for backtracking.
type Step[S State] struct {
	State S

	cost     int
	parent   *Step[S]
	children []*Step[S]
}

// Backtrack returns the path of steps from the root to this step.
func (s *Step[S]) Backtrack() (steps []*Step[S]) {
	if s == nil {
		return nil
	}

	for steps = append(steps, s); s.parent != nil; s = s.parent {
		steps = append(steps, s.parent)
	}
	slices.Reverse(steps)
	return steps
}

// Cost returns the accumulated step cost.
func (s *Step[S]) Cost() int { return s.cost }

func (s *Step[S]) visited(key string) bool {
	return key == s.State.Key() || (s.parent != nil && s.parent.visited(key))
}

// NewBruter creates a new bruter.
func NewBruter[S State](processor func(S) []S) *Bruter[S] {
	return &Bruter[S]{
		steps:   make(map[string]*Step[S]),
		process: processor,
	}
}

// Bruter searches a state space for a Done state using a caller-supplied successor function.
type Bruter[S State] struct {
	steps map[string]*Step[S]

	process func(S) []S // process state to next state
}

// Find searches for a done state using the given method.
func (b Bruter[S]) Find(state S, method METHOD) (finalStep *Step[S], err error) {
	if err := state.Preprocess(); err != nil {
		return nil, err
	}
	switch method {
	case DFS:
		return b.dfs(NewStep[S](state, nil)), nil
	case BFS:
		return b.bfs(NewStep[S](state, nil)), nil
	default:
		return nil, errors.New("unknown method")
	}
}

func (b Bruter[S]) dfs(s *Step[S]) (finalStep *Step[S]) {
	for _, nextState := range b.process(s.State) {
		key := nextState.Key()

		if s.visited(key) || b.steps[key] != nil {
			continue
		}

		nextStep := NewStep(nextState, s)
		b.steps[key] = nextStep
		s.children = append(s.children, nextStep)

		if nextState.Done() {
			return nextStep
		}

		if step := b.dfs(nextStep); step != nil {
			return step
		}
	}
	return nil
}

func (b Bruter[S]) bfs(s *Step[S]) (finalStep *Step[S]) {
	var queue Queue[S]
	queue.Enqueue(s)
	for !queue.Empty() {
		var steps []*Step[S]

		s = queue.Dequeue()
		for _, nextState := range b.process(s.State) {
			key := nextState.Key()

			if s.visited(key) || b.steps[key] != nil {
				continue
			}

			nextStep := NewStep(nextState, s)
			b.steps[key] = nextStep
			s.children = append(s.children, nextStep)

			if nextState.Done() {
				return nextStep
			}

			steps = append(steps, nextStep)
		}
		queue.Enqueue(steps...)
	}
	return nil
}

// Queue is a FIFO queue of search steps.
type Queue[S State] struct {
	queue []*Step[S]
}

// Empty reports whether the queue has no items.
func (q *Queue[S]) Empty() bool { return len(q.queue) == 0 }

// Enqueue appends steps to the queue.
func (q *Queue[S]) Enqueue(steps ...*Step[S]) {
	q.queue = append(q.queue, steps...)
}

// Dequeue removes and returns the head of the queue.
func (q *Queue[S]) Dequeue() *Step[S] {
	if len(q.queue) == 0 {
		return nil
	}

	s := q.queue[0]
	q.queue = q.queue[1:]
	return s
}
