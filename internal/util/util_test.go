package util_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"app/internal/util"
)

func TestEscapeInvisibleStr(t *testing.T) {
	require.Equal(t, "\\u200E\t\\u200B", util.EscapeInvisible("\u200e\t\u200b"))
	require.Equal(t, "你好", util.EscapeInvisible("你好"))
}
