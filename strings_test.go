package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEscapeInvisibleStr(t *testing.T) {
	require.Equal(t, "\\u200e\t\\u200b", EscapeInvisible("\u200e\t\u200b"))
	require.Equal(t, "你好", EscapeInvisible("你好"))
}
