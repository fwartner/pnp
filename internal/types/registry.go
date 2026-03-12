package types

import (
	"fmt"
	"sort"
)

var registry = make(map[string]ProjectType)

// Register adds a project type to the registry. Panics on duplicate.
func Register(pt ProjectType) {
	name := pt.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("project type %q already registered", name))
	}
	registry[name] = pt
}

// Get returns the project type for the given name, or nil if not found.
func Get(name string) ProjectType {
	return registry[name]
}

// All returns all registered project types, sorted by name.
func All() []ProjectType {
	pts := make([]ProjectType, 0, len(registry))
	for _, pt := range registry {
		pts = append(pts, pt)
	}
	sort.Slice(pts, func(i, j int) bool {
		return pts[i].Name() < pts[j].Name()
	})
	return pts
}

// Names returns all registered type names, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Detect runs detection across all registered types and returns the best match.
// Returns the detected type and confidence level, or nil and "" if nothing detected.
func Detect(dir string) (ProjectType, string) {
	confidenceOrder := map[string]int{"high": 3, "medium": 2, "low": 1, "": 0}

	var bestType ProjectType
	bestConfidence := ""

	for _, pt := range registry {
		conf := pt.Detect(dir)
		if confidenceOrder[conf] > confidenceOrder[bestConfidence] {
			bestType = pt
			bestConfidence = conf
		}
	}

	return bestType, bestConfidence
}
