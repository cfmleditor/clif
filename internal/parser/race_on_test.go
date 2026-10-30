//go:build race

package parser

// raceEnabled is true in a -race build, where sync.Pool drops what it holds
// at random and allocation counts are not the ones a release build makes.
const raceEnabled = true
