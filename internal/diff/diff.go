package diff

import (
	"fmt"
	"strings"

	"github.com/aymanbagabas/go-udiff"

	"app/internal/myers"
	"app/internal/util"
)

// DefaultContextLines is the number of unchanged lines shown around each hunk.
const DefaultContextLines = 3

// FullContext makes Diff include every unchanged line of the content.
const FullContext = -1

func Diff(name, before, after string, contextLines int) string {
	before = strings.ReplaceAll(before, "\r\n", "\n")
	after = strings.ReplaceAll(after, "\r\n", "\n")

	before = util.EscapeInvisible(before)
	after = util.EscapeInvisible(after)

	if contextLines < 0 {
		// udiff multiplies this by 2 internally, so an unbounded value would
		// overflow; the number of lines is already enough to show everything.
		contextLines = strings.Count(before, "\n") + strings.Count(after, "\n") + 1
	}

	edits := myers.ComputeEdits(before, after)
	unified, err := udiff.ToUnified(name, name, before, edits, contextLines)
	if err != nil {
		// Can't happen: edits are consistent.
		panic(fmt.Sprintf("internal error in diff.Unified: %v", err))
	}

	return unified
}
