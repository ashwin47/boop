// Package levels defines event severity levels.
package levels

// Levels, in ascending severity.
const (
	Info     = "info"
	Success  = "success"
	Warning  = "warning"
	Error    = "error"
	Critical = "critical"
)

// All lists the levels in ascending severity.
var All = []string{Info, Success, Warning, Error, Critical}

var rank = map[string]int{Info: 0, Success: 1, Warning: 2, Error: 3, Critical: 4}

// Valid reports whether l is a known level.
func Valid(l string) bool {
	_, ok := rank[l]
	return ok
}

// AtLeast reports whether l is at least as severe as min. Unknown values rank lowest.
func AtLeast(l, min string) bool {
	return rank[l] >= rank[min]
}
