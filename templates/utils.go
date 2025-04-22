package templates

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
)

func setQuery(u *url.URL, key string, value any) templ.SafeURL {
	q := u.Query()

	q.Set(key, fmt.Sprint(value))

	return templ.SafeURL(u.Path + "?" + q.Encode())
}

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Precompile the regular expression for efficiency.
// Note: Go's raw string literals (` “ `) handle backslashes differently than Python's r"".
// The pattern seems compatible, but ensure escaping is correct if needed.
// The original Python pattern excludes some CJK punctuation and whitespace in the path.
var urlPattern = regexp.MustCompile(`(?i)(https?://[^/]+(?:/[^（），。() \r\n\s]*)?)`)

// AutoURL finds URLs in a string and replaces them with HTML anchor tags.
// Non-URL parts of the string are HTML-escaped.
// It returns template.HTML to indicate the result is safe HTML for Go templates.
func AutoURL(s string) string {
	var builder strings.Builder
	lastIndex := 0

	// Find all matches and their indices.
	matches := urlPattern.FindAllStringIndex(s, -1)

	for _, matchIndices := range matches {
		startIndex := matchIndices[0]
		endIndex := matchIndices[1]
		u := s[startIndex:endIndex]

		// Append escaped text before the URL.
		builder.WriteString(html.EscapeString(s[lastIndex:startIndex]))

		// Append the anchor tag for the URL.
		escapedURL := html.EscapeString(u)
		builder.WriteString(fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, escapedURL, escapedURL))

		lastIndex = endIndex
	}

	// Append any remaining escaped text after the last URL.
	if lastIndex < len(s) {
		builder.WriteString(html.EscapeString(s[lastIndex:]))
	}

	// Return the result as safe HTML.
	return builder.String()
}

func relativeTime(now time.Time, t time.Time) string {
	d := now.Sub(t).Truncate(time.Second)

	// Handle future times if necessary, for now assume t <= now
	// and return the duration magnitude.
	if d < 0 {
		d = -d
	}

	// Handle very small durations
	if d < time.Second {
		return "just now"
	}

	if d < time.Minute {
		// Format: Xs
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}

	if d < time.Hour {
		// Format: XmYs or Xm
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm ago", m)
		}
		return fmt.Sprintf("%dm%ds ago", m, s)
	}

	const day = 24 * time.Hour
	if d < day {
		// Format: XhYm or Xh
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh ago", h)
		}
		return fmt.Sprintf("%dh%dm ago", h, m)
	}

	// Using 365 days for a year is an approximation
	const year = 365 * day

	if d < year {
		// Format: Xday Yh / Xdays Yh or Xday / Xdays
		days := int(d / day)

		dayStr := "day"
		if days > 1 {
			dayStr = "days"
		}

		return fmt.Sprintf("%d%s ago", days, dayStr)
	}

	// Format: XyYd or Xy
	years := int(d / year)
	days := int((d % year) / day) // Days remaining after full years

	if days == 0 {
		return fmt.Sprintf("%dy ago", years)
	}
	return fmt.Sprintf("%dy%dd ago", years, days)
}
