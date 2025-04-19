package templates

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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

// Example usage (optional, for demonstration)
// func main() {
// 	text := "Check out this link: http://example.com and also https://example.org/page?q=test. Don't forget (http://example.net)."
// 	htmlResult := AutoURL(text)
// 	fmt.Println(htmlResult)
// 	// Output: Check out this link: &lt;a href="http://example.com" target="_blank"&gt;http://example.com&lt;/a&gt; and also &lt;a href="https://example.org/page?q=test" target="_blank"&gt;https://example.org/page?q=test&lt;/a&gt;. Don&#39;t forget (&lt;a href="http://example.net" target="_blank"&gt;http://example.net&lt;/a&gt;).
// }
