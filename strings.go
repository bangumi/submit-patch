package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Note: Go's \p{Z} might differ slightly from Python's re2 \p{Z}.
// Go's regexp uses RE2 syntax. Python's re2 wrapper also uses RE2. They should be compatible.
// Pattern: Match characters NOT in the set {tab, cr, lf, Letter, Mark, Number, Punctuation, Symbol, Separator}
var invisiblePattern = regexp.MustCompile(`[^\t\r\n\p{L}\p{M}\p{N}\p{P}\p{S}\p{Z}]`)

// CheckInvalidInputStr checks if any of the input strings contain invalid characters (non-printable, control chars).
// It checks line by line. If an invalid character is found, it returns an error.
// This mimics the behavior of the Python version, including line-by-line checking.
func CheckInvalidInputStr(ss ...string) error {
	for _, s := range ss {
		scanner := bufio.NewScanner(strings.NewReader(s))
		for scanner.Scan() {
			line := scanner.Text()
			match := invisiblePattern.FindStringSubmatch(line)
			if len(match) > 0 {
				// Use %q to safely quote the character and the line
				return fmt.Errorf("invalid character %q in line %q", match[0], line)
			}
		}
		if err := scanner.Err(); err != nil {
			// Handle potential scanner errors, though unlikely for strings.Reader
			return fmt.Errorf("error scanning string: %w", err)
		}
	}
	return nil // No invalid characters found
}

// ContainsInvalidInputStr checks if any of the input strings contain invalid characters.
// If found, it returns the first occurrence of such a character. Otherwise, returns an empty string.
func ContainsInvalidInputStr(ss ...string) string {
	for _, s := range ss {
		match := invisiblePattern.FindString(s)
		if match != "" {
			return match
		}
	}
	return ""
}

// replInvisible converts a matched invisible character sequence to its Unicode escape representation.
// This mimics the Python `__repl` function using `encode("unicode-escape").decode()`.
func replInvisible(matchedInvisible string) string {
	var sb strings.Builder
	for _, r := range matchedInvisible {
		// strconv.QuoteRuneToASCII produces 'X' style quotes (e.g., '\x00', '\u1234').
		// We remove the surrounding single quotes to match the Python output.
		quoted := strconv.QuoteRuneToASCII(r)
		if len(quoted) >= 3 && quoted[0] == '\'' && quoted[len(quoted)-1] == '\'' {
			sb.WriteString(quoted[1 : len(quoted)-1])
		} else {
			// Fallback if QuoteRuneToASCII behaves unexpectedly.
			// This path is unlikely for characters matched by invisiblePattern.
			sb.WriteString(quoted)
		}
	}
	return sb.String()
}

// EscapeInvisible replaces all invisible characters in a string with their Unicode escape sequences
// (e.g., \x00, \uABCD).
func EscapeInvisible(s string) string {
	return invisiblePattern.ReplaceAllStringFunc(s, replInvisible)
}
