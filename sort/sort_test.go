package sort_test

import (
	"fmt"
	"testing"

	"github.com/tr1v3r/pkg/sort"
)

// A couple of type definitions to make the units clear.
type earthMass float64
type au float64

// A Planet defines the properties of a solar system object.
type Planet struct {
	name     string
	mass     earthMass
	distance au
}

var planets = []Planet{
	{"Mercury", 0.055, 0.4},
	{"Venus", 0.815, 0.7},
	{"Earth", 1.0, 1.0},
	{"Mars", 0.107, 1.5},
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
	fmt.Println("By name:", planets)

	sort.By[Planet](mass).Sort(planets)
	fmt.Println("By mass:", planets)

	sort.By[Planet](distance).Sort(planets)
	fmt.Println("By distance:", planets)

	sort.By[Planet](decreasingDistance).Sort(planets)
	fmt.Println("By decreasing distance:", planets)

}

// A Change is a record of source code changes, recording user, language, and delta size.
type Change struct {
	user     string
	language string
	lines    int
}

var changes = []Change{
	{"gri", "Go", 100},
	{"ken", "C", 150},
	{"glenda", "Go", 200},
	{"rsc", "Go", 200},
	{"r", "Go", 100},
	{"ken", "Go", 200},
	{"dmr", "C", 100},
	{"r", "C", 150},
	{"gri", "Smalltalk", 80},
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
	fmt.Println("By user:", changes)

	// More examples.
	sort.MultiBy(user, increasingLines).Sort(changes)
	fmt.Println("By user,<lines:", changes)

	sort.MultiBy(user, decreasingLines).Sort(changes)
	fmt.Println("By user,>lines:", changes)

	sort.MultiBy(language, increasingLines).Sort(changes)
	fmt.Println("By language,<lines:", changes)

	sort.MultiBy(language, increasingLines, user).Sort(changes)
	fmt.Println("By language,<lines,user:", changes)

}
