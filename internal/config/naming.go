package config

import "strings"

// ScopePrefixedName constructs a scope-prefixed name from scope and short name.
// Example: ScopePrefixedName("agency", "pixel-process") → "agency-pixel-process"
func ScopePrefixedName(scope, shortName string) string {
	return scope + "-" + shortName
}

// ShortName extracts the name without scope prefix.
// Example: ShortName("agency-pixel-process", "agency") → "pixel-process"
func ShortName(fullName, scope string) string {
	prefix := scope + "-"
	if strings.HasPrefix(fullName, prefix) {
		return strings.TrimPrefix(fullName, prefix)
	}
	return fullName
}

// HasScopePrefix checks if a name already has the scope prefix.
func HasScopePrefix(name, scope string) bool {
	return strings.HasPrefix(name, scope+"-")
}
