//go:build !race

package examples

// raceEnabled avoids race instrumentation in ordinary example runs.
const raceEnabled = false
