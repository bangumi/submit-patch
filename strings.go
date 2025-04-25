package main

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Note: Go's \p{Z} might differ slightly from Python's re2 \p{Z}.
// Go's regexp uses RE2 syntax. Python's re2 wrapper also uses RE2. They should be compatible.
// Pattern: Match characters NOT in the set {tab, cr, lf, Letter, Mark, Number, Punctuation, Symbol, Separator}
var invisiblePattern = regexp.MustCompile(`[^\t\r\n\p{L}\p{M}\p{N}\p{P}\p{S}\p{Z}]`)

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
	return nil
}

func ContainsInvalidInputStr(ss ...string) string {
	for _, s := range ss {
		match := invisiblePattern.FindString(s)
		if match != "" {
			return match
		}
	}
	return ""
}

func replInvisible(matchedInvisible string) string {
	var sb = bytes.NewBuffer(make([]byte, 0, 8))
	for _, r := range matchedInvisible {
		if !isValidUnicodeCodePoint(r) {
			fmt.Fprintf(sb, "\\x%02X", r)
			continue
		}
		if r <= 0xFFFF {
			fmt.Fprintf(sb, "\\u%04X", r)
		} else {
			fmt.Fprintf(sb, "\\U%08X", r)
		}
	}
	return sb.String()
}

func isValidUnicodeCodePoint(r rune) bool {
	return r != utf8.RuneError && !(r >= 0xD800 && r <= 0xDFFF) && r <= 0x10FFFF
}

func EscapeInvisible(s string) string {
	return invisiblePattern.ReplaceAllStringFunc(s, replInvisible)
}
