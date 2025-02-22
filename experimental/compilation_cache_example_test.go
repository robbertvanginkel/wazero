package experimental_test

import (
	"context"
	"log"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental"
)

// This is a basic example of using the file system compilation cache via WithCompilationCacheDirName.
// The main goal is to show how it is configured.
func Example_withCompilationCacheDirName() {
	// Prepare a cache directory.
	cacheDir, err := os.MkdirTemp("", "example")
	if err != nil {
		log.Panicln(err)
	}
	defer os.RemoveAll(cacheDir)

	// Append the directory into the context for configuration.
	ctx, err := experimental.WithCompilationCacheDirName(context.Background(), cacheDir)
	if err != nil {
		log.Panicln(err)
	}

	// Repeat newRuntimeCompileClose with the same cache directory.
	newRuntimeCompileClose(ctx)
	// Since the above stored compiled functions to dist, below won't compile.
	// Instead, code stored in the file cache is re-used.
	newRuntimeCompileClose(ctx)
	newRuntimeCompileClose(ctx)

	// Output:
	//
}

// newRuntimeCompileDestroy creates a new wazero.Runtime, compile a binary, and then delete the runtime.
func newRuntimeCompileClose(ctx context.Context) {
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx) // This closes everything this Runtime created except the file system cache.

	_, err := r.CompileModule(ctx, fsWasm)
	if err != nil {
		log.Panicln(err)
	}
}
