package templates

import (
	"fmt"
	"net/url"
	"strconv"

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
