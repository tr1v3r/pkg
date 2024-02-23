package brute

import (
	"errors"
	"slices"
)

type METHOD string

const (
	BFS METHOD = "bfs"
	DFS METHOD = "dfs"
)

type State interface {
	Key() string
	Done() bool
	Preprocess() error
}

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

type Step[S State] struct {
	State S

	cost     int
	parent   *Step[S]
	children []*Step[S]
}

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
func (s *Step[S]) Cost() int { return s.cost }

func (s *Step[S]) visited(key string) bool {
	return key == s.State.Key() || (s.parent != nil && s.parent.visited(key))
}

func NewBruter[S State](processor func(S) []S) *Bruter[S] {
	return &Bruter[S]{
		steps:   make(map[string]*Step[S]),
		process: processor,
	}
}

type Bruter[S State] struct {
	steps map[string]*Step[S]

	process func(S) []S // process state to next state
}

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

type Queue[S State] struct {
	queue []*Step[S]
}

func (q *Queue[S]) Empty() bool { return len(q.queue) == 0 }
func (q *Queue[S]) Enqueue(steps ...*Step[S]) {
	q.queue = append(q.queue, steps...)
}
func (q *Queue[S]) Dequeue() *Step[S] {
	if len(q.queue) == 0 {
		return nil
	}

	s := q.queue[0]
	q.queue = q.queue[1:]
	return s
}
