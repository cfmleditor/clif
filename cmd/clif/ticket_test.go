package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// oracleTicket is the expression ticketIn replaced, kept as the oracle it is
// held to.
var oracleTicket = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)

var ticketCases = []string{
	"ABC-123", "feature/ABC-123-add-thing", "ABC-123: fix", "fix ABC-123", "AB-1", "A-1", "ABC-", "ABC-x", "-123", "ABC123", "abc-123", "Abc-123",
	"xABC-123", "ABC-123x", "ABC-123_", "_ABC-123", "ABC_-123", "ABC-12-34", "ABC-12 DEF-34", "A1-2", "A1B2-3", "1A-2", "AB--1", "AB-1-2", "ABC-123.", ".ABC-123",
	"ABc-123", "aBC-123 DEF-9", "ABC- 123", "ABC -123", "ABC-1 2", "x ABC-1", "ABC-1/", "/ABC-1", "(ABC-1)", "[ABC-1]", "ABC-1\n", "\nABC-1", "é ABC-1", "ABC-1é", "éABC-1",
	"ABC-99999999999999999999", "AAAAAAAAAAAAAAAA-1", "", " ", "main", "release/1.2.3", "HEAD", "wip",
}

var ticketFragments = []string{"A", "B", "Z", "a", "z", "0", "1", "9", "-", "_", " ", "/", ".", "\n", "é", "ABC", "AB-1", "x", "(", ")"}

func randomTicketText(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := ticketCases[rng.IntN(len(ticketCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + ticketFragments[rng.IntN(len(ticketFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + ticketFragments[rng.IntN(len(ticketFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(12) {
		b.WriteString(ticketFragments[rng.IntN(len(ticketFragments))])
	}

	return b.String()
}

func ticketDisagreement(s string) string {
	got, ok := ticketIn(s)
	want := oracleTicket.FindString(s)

	if ok != oracleTicket.MatchString(s) || got != want {
		return fmt.Sprintf("ticketIn(%q) = %q %v, want %q %v", s, got, ok, want, oracleTicket.MatchString(s))
	}

	return ""
}

// ticketIn finds the issue key the expression did, on branch names and commit
// subjects and texts built from fragments of them; one text in ten holds a key.
func TestTicketInMatchesItsExpression(t *testing.T) {
	n := 300_000
	if testing.Short() {
		n = 30_000
	}

	texts := append([]string(nil), ticketCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomTicketText(rng))
	}

	failures, found := 0, 0

	for _, s := range texts {
		if msg := ticketDisagreement(s); msg != "" {
			t.Error(msg)

			failures++

			if failures == 10 {
				return
			}
		}

		if oracleTicket.MatchString(s) {
			found++
		}
	}

	if found < len(texts)/10 {
		t.Errorf("%d of %d texts hold a key; the generated texts do not reach the reader", found, len(texts))
	}
}

// FuzzTicketIn holds ticketIn to its expression on any text.
func FuzzTicketIn(f *testing.F) {
	for _, s := range ticketCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if msg := ticketDisagreement(s); msg != "" {
			t.Error(msg)
		}
	})
}
