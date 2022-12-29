package gojs_test

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func Test_goroutine(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := compileAndRun(testCtx, "goroutine", wazero.NewModuleConfig())

	require.EqualError(t, err, `module "" closed with exit_code(0)`)
	require.Zero(t, stderr)
	require.Equal(t, `producer
consumer
`, stdout)
}

func Test_mem(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := compileAndRun(testCtx, "mem", wazero.NewModuleConfig())

	require.EqualError(t, err, `module "" closed with exit_code(0)`)
	require.Zero(t, stderr)
	require.Zero(t, stdout)
}

func Test_stdio(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := compileAndRun(testCtx, "stdio", wazero.NewModuleConfig().
		WithStdin(strings.NewReader("stdin\n")))

	require.EqualError(t, err, `module "" closed with exit_code(0)`)
	require.Equal(t, "println stdin\nStderr.Write", stderr)
	require.Equal(t, "Stdout.Write", stdout)
}

func Test_gc(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := compileAndRun(testCtx, "gc", wazero.NewModuleConfig())

	require.EqualError(t, err, `module "" closed with exit_code(0)`)
	require.Equal(t, "", stderr)
	require.Equal(t, "before gc\nafter gc\n", stdout)
}

func Test_largeio(t *testing.T) {
	t.Parallel()

	largeStdin := make([]byte, 2*1024*1024) // 2 mb
	stdout, stderr, err := compileAndRun(testCtx, "largeio", wazero.NewModuleConfig().
		WithStdin(bytes.NewReader(largeStdin)))

	require.Equal(t, "", stderr)
	require.EqualError(t, err, `module "" closed with exit_code(0)`)
	require.Equal(t, strconv.Itoa(len(largeStdin))+"\n", stdout)
}
