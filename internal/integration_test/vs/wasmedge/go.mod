module github.com/tetratelabs/wazero/internal/integration_test/vs/wasmedge

go 1.17

require (
	github.com/second-state/WasmEdge-go v0.11.1
	github.com/tetratelabs/wazero v0.0.0
)

replace github.com/tetratelabs/wazero => ../../../..
