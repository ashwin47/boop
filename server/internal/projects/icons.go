package projects

import (
	"hash/fnv"
	"strings"
)

// Project icons are abstract shapes from the design palette, encoded as
// "<shape>:<color>" (for example "triangle:mint"). Clients render them
// natively. Short free text (an emoji) is still accepted for compatibility.

// IconShapes lists the supported shapes.
var IconShapes = []string{"circle", "ring", "square", "diamond", "triangle", "hexagon", "pill", "blob"}

// IconColors lists the supported palette names with their hex values.
var IconColors = []struct{ Name, Hex string }{
	{"periwinkle", "#7C83E8"},
	{"mint", "#5FBF9F"},
	{"blush", "#E88CB0"},
	{"amber", "#E8B34C"},
	{"violet", "#9B7BEA"},
	{"slate", "#9C9EAB"},
}

// ValidIcon reports whether s is empty, a known shape:color pair, or short free text.
func ValidIcon(s string) bool {
	if s == "" {
		return true
	}
	if shape, color, ok := strings.Cut(s, ":"); ok {
		return contains(IconShapes, shape) && iconColor(color)
	}
	return len([]rune(s)) <= 4
}

// DefaultIcon picks a deterministic shape:color for a seed (the project slug),
// so every project has a distinct glyph without the user choosing one.
func DefaultIcon(seed string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	n := h.Sum32()
	shape := IconShapes[n%uint32(len(IconShapes))]
	// Skip slate for defaults; it reads as "disabled".
	color := IconColors[(n/uint32(len(IconShapes)))%uint32(len(IconColors)-1)].Name
	return shape + ":" + color
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func iconColor(name string) bool {
	for _, c := range IconColors {
		if c.Name == name {
			return true
		}
	}
	return false
}
