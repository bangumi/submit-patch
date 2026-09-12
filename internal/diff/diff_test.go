package diff_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"app/internal/diff"
)

func TestDiffContextLines(t *testing.T) {
	before := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	after := "1\n2\n3\n4\nX\n6\n7\n8\n9\n10\n"

	partial := diff.Diff("x", before, after, diff.DefaultContextLines)
	require.Contains(t, partial, " 2\n")
	require.NotContains(t, partial, " 1\n")
	require.NotContains(t, partial, " 9\n")
	require.Contains(t, partial, "-5\n+X\n")

	full := diff.Diff("x", before, after, diff.FullContext)
	require.Contains(t, full, " 1\n")
	require.Contains(t, full, " 9\n")
	require.Contains(t, full, " 10\n")
	require.Contains(t, full, "-5\n+X\n")
}
