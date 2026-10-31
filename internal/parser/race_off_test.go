//go:build !race

package parser

// raceEnabled is true in a -race build; see race_on_test.go.
const raceEnabled = false
