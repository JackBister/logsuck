package search

type Search struct {
	Fragments    map[string]struct{}
	NotFragments map[string]struct{}
	Fields       map[string][]string
	NotFields    map[string][]string
	Sources      map[string]struct{}
	NotSources   map[string]struct{}
	Hosts        map[string]struct{}
	NotHosts     map[string]struct{}
}
