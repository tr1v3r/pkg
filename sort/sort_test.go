package sort_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/tr1v3r/pkg/sort"
)

// A couple of type definitions to make the units clear.
type earthMass float64
type au float64

// A Planet defines the properties of a solar system object.
type Planet struct {
	id       int
	name     string
	mass     earthMass
	distance au
}

func (p Planet) ID() int { return p.id }

var planets = []Planet{
	{1, "Mercury", 0.055, 0.4},
	{2, "Venus", 0.815, 0.7},
	{3, "Earth", 1.0, 1.0},
	{4, "Mars", 0.107, 1.5},
}

// ExampleSortKeys demonstrates a technique for sorting a struct type using programmable sort criteria.
func Test_SortKeys(t *testing.T) {
	// Closures that order the Planet structure.
	name := func(p1, p2 *Planet) bool { return p1.name < p2.name }
	mass := func(p1, p2 *Planet) bool { return p1.mass < p2.mass }
	distance := func(p1, p2 *Planet) bool { return p1.distance < p2.distance }
	decreasingDistance := func(p1, p2 *Planet) bool { return distance(p2, p1) }

	// Sort the planets by the various criteria.
	sort.By[Planet](name).Sort(planets)
	if order := []int{3, 4, 1, 2}; !matchIDSort(planets, order...) {
		t.Errorf("mismatch sort result by name, expected: %+v, got: %+v", order, toIDSlice(planets))
	}
	t.Log("By name:", planets)

	sort.By[Planet](mass).Sort(planets)
	if order := []int{1, 4, 2, 3}; !matchIDSort(planets, order...) {
		t.Errorf("mismatch sort result by mass, expected: %+v, got: %+v", order, toIDSlice(planets))
	}
	t.Log("By mass:", planets)

	sort.By[Planet](distance).Sort(planets)
	if order := []int{1, 2, 3, 4}; !matchIDSort(planets, order...) {
		t.Errorf("mismatch sort result by distance, expected: %+v, got: %+v", order, toIDSlice(planets))
	}
	t.Log("By distance:", planets)

	sort.By[Planet](decreasingDistance).Sort(planets)
	if order := []int{4, 3, 2, 1}; !matchIDSort(planets, order...) {
		t.Errorf("mismatch sort result by distance desc, expected: %+v, got: %+v", order, toIDSlice(planets))
	}
	t.Log("By decreasing distance:", planets)

}

// A Change is a record of source code changes, recording user, language, and delta size.
type Change struct {
	id       int
	user     string
	language string
	lines    int
}

func (c Change) ID() int { return c.id }

var changes = []Change{
	{1, "gri", "Go", 100},
	{2, "ken", "C", 150},
	{3, "glenda", "Go", 200},
	{4, "rsc", "Go", 200},
	{5, "r", "Go", 100},
	{6, "ken", "Go", 200},
	{7, "dmr", "C", 100},
	{8, "r", "C", 150},
	{9, "gri", "Smalltalk", 80},
}

// ExampleMultiKeys demonstrates a technique for sorting a struct type using different
// sets of multiple fields in the comparison. We chain together "Less" functions, each of
// which compares a single field.
func Test_SortMultiKeys(t *testing.T) {
	// Closures that order the Change structure.
	user := func(c1, c2 *Change) bool { return c1.user < c2.user }
	language := func(c1, c2 *Change) bool { return c1.language < c2.language }
	increasingLines := func(c1, c2 *Change) bool { return c1.lines < c2.lines }
	decreasingLines := func(c1, c2 *Change) bool { return c1.lines > c2.lines } // Note: > orders downwards.

	// Simple use: Sort by user.
	sort.MultiBy[Change](user).Sort(changes)
	if order := []int{7, 3, 1, 9, 2, 6, 5, 8, 4}; !matchIDSort(changes, order...) {
		t.Errorf("mismatch sort result by user, expected: %+v, got: %+v", order, toIDSlice(changes))
	}
	t.Log("By user:", changes)

	// More examples.
	sort.MultiBy(user, increasingLines).Sort(changes)
	if order := []int{7, 3, 9, 1, 2, 6, 5, 8, 4}; !matchIDSort(changes, order...) {
		t.Errorf("mismatch sort result by user,<lines , expected: %+v, got: %+v", order, toIDSlice(changes))
	}
	t.Log("By user,<lines:", changes)

	sort.MultiBy(user, decreasingLines).Sort(changes)
	if order := []int{7, 3, 1, 9, 6, 2, 8, 5, 4}; !matchIDSort(changes, order...) {
		t.Errorf("mismatch sort result by user,>lines , expected: %+v, got: %+v", order, toIDSlice(changes))
	}
	t.Log("By user,>lines:", changes)

	sort.MultiBy(language, increasingLines).Sort(changes)
	if order := []int{7, 2, 8, 1, 5, 3, 6, 4, 9}; !matchIDSort(changes, order...) {
		t.Errorf("mismatch sort result by language,<lines , expected: %+v, got: %+v", order, toIDSlice(changes))
	}
	t.Log("By language,<lines:", changes)

	sort.MultiBy(language, increasingLines, user).Sort(changes)
	if order := []int{7, 2, 8, 1, 5, 3, 6, 4, 9}; !matchIDSort(changes, order...) {
		t.Errorf("mismatch sort result by language,<lines,user , expected: %+v, got: %+v", order, toIDSlice(changes))
	}
	t.Log("By language,<lines,user:", changes)

}

type Item interface{ ID() int }

func matchIDSort(tgt any, order ...int) bool {
	t := reflect.ValueOf(tgt)

	if t.Kind() != reflect.Slice {
		panic(fmt.Errorf("invalid params: tgt must be a slice"))
	}

	if t.Len() != len(order) {
		return false
	}

	for index, id := range order {
		if t.Index(index).Interface().(Item).ID() != id {
			return false
		}
	}
	return true
}

func toIDSlice(tgt any) (ret []int) {
	t := reflect.ValueOf(tgt)

	if t.Kind() != reflect.Slice {
		panic(fmt.Errorf("invalid params: tgt must be a slice"))
	}

	for i := 0; i < t.Len(); i++ {
		ret = append(ret, t.Index(i).Interface().(Item).ID())
	}
	return ret
}
