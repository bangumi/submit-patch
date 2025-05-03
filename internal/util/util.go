package util

import (
	"bytes"
	"fmt"
	"regexp"
	"unicode/utf8"
)

var invisiblePattern = regexp.MustCompile(`[^\t\r\n\p{L}\p{M}\p{N}\p{P}\p{S}\p{Z}]`)

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
