package sort

import (
	"sort"
)

// ============ single sort ============

// By is the type of a "less" function that defines the ordering of its T arguments.
type By[T any] func(l, r *T) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By[T]) Sort(items []*T) {
	sort.Sort(&sorter[T]{
		items: items,
		by:    by, // The Sort method's receiver is the function (closure) that defines the sort order.
	})
}

// sorter joins a By function and a slice of T to be sorted.
type sorter[T any] struct {
	items []*T
	by    By[T] // Closure used in the Less method.
}

func (s *sorter[T]) Len() int           { return len(s.items) }
func (s *sorter[T]) Swap(i, j int)      { s.items[i], s.items[j] = s.items[j], s.items[i] }
func (s *sorter[T]) Less(i, j int) bool { return s.by(s.items[i], s.items[j]) }

// ReverseBy return an reverse closure By
func ReverseBy[T any](by By[T]) By[T] { return func(l, r *T) bool { return by(r, l) } }

// ============ multi sort ============

// MultiBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func MultiBy[T any](less ...By[T]) *multiSorter[T] { return &multiSorter[T]{less: less} }

type multiSorter[T any] struct {
	items []*T
	less  []By[T]
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (ms *multiSorter[T]) Sort(items []*T) {
	ms.items = items
	sort.Sort(ms)
}

// implement of sort.Interface

func (ms *multiSorter[T]) Len() int      { return len(ms.items) }
func (ms *multiSorter[T]) Swap(i, j int) { ms.items[i], ms.items[j] = ms.items[j], ms.items[i] }

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that discriminates between
// the two items (one is less than the other).
func (ms *multiSorter[T]) Less(i, j int) bool {
	l, r := ms.items[i], ms.items[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(l, r):
			// l < r, so we have a decision.
			return true
		case less(r, l):
			// l > r, so we have a decision.
			return false
		}
		// l == r; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](l, r)
}
