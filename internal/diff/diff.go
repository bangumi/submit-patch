package diff

import (
	"fmt"

	"github.com/aymanbagabas/go-udiff"

	"app/internal/myers"
	"app/internal/util"
)

func Diff(name, before, after string) string {
	before = util.EscapeInvisible(before)
	after = util.EscapeInvisible(after)

	edits := myers.ComputeEdits(before, after)
	unified, err := udiff.ToUnified(name, name, before, edits, 3)
	if err != nil {
		// Can't happen: edits are consistent.
		panic(fmt.Sprintf("internal error in diff.Unified: %v", err))
	}

	return unified
}
