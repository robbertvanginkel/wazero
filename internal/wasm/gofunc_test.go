package wasm

import (
	"context"
	"math"
	"testing"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

// testCtx is an arbitrary, non-default context. Non-nil also prevents linter errors.
var testCtx = context.WithValue(context.Background(), struct{}{}, "arbitrary")

func Test_parseGoFunc(t *testing.T) {
	var tests = []struct {
		name              string
		input             interface{}
		expectNeedsModule bool
		expectedType      *FunctionType
	}{
		{
			name:         "() -> ()",
			input:        func() {},
			expectedType: &FunctionType{},
		},
		{
			name:         "(ctx) -> ()",
			input:        func(context.Context) {},
			expectedType: &FunctionType{},
		},
		{
			name:              "(ctx, mod) -> ()",
			input:             func(context.Context, api.Module) {},
			expectNeedsModule: true,
			expectedType:      &FunctionType{},
		},
		{
			name:         "all supported params and i32 result",
			input:        func(uint32, uint64, float32, float64, uintptr) uint32 { return 0 },
			expectedType: &FunctionType{Params: []ValueType{i32, i64, f32, f64, externref}, Results: []ValueType{i32}},
		},
		{
			name:         "all supported params and i32 result - (ctx)",
			input:        func(context.Context, uint32, uint64, float32, float64, uintptr) uint32 { return 0 },
			expectedType: &FunctionType{Params: []ValueType{i32, i64, f32, f64, externref}, Results: []ValueType{i32}},
		},
		{
			name:              "all supported params and i32 result - (ctx, mod)",
			input:             func(context.Context, api.Module, uint32, uint64, float32, float64, uintptr) uint32 { return 0 },
			expectNeedsModule: true,
			expectedType:      &FunctionType{Params: []ValueType{i32, i64, f32, f64, externref}, Results: []ValueType{i32}},
		},
	}
	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			paramTypes, resultTypes, code, err := parseGoReflectFunc(tc.input)
			require.NoError(t, err)
			_, isModuleFunc := code.GoFunc.(api.GoModuleFunction)
			require.Equal(t, tc.expectNeedsModule, isModuleFunc)
			require.Equal(t, tc.expectedType, &FunctionType{Params: paramTypes, Results: resultTypes})
		})
	}
}

func Test_parseGoFunc_Errors(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectedErr string
	}{
		{
			name:        "module no context",
			input:       func(api.Module) {},
			expectedErr: "invalid signature: api.Module parameter must be preceded by context.Context",
		},
		{
			name:        "not a func",
			input:       struct{}{},
			expectedErr: "kind != func: struct",
		},
		{
			name:        "unsupported param",
			input:       func(context.Context, uint32, string) {},
			expectedErr: "param[2] is unsupported: string",
		},
		{
			name:        "unsupported result",
			input:       func() string { return "" },
			expectedErr: "result[0] is unsupported: string",
		},
		{
			name:        "error result",
			input:       func() error { return nil },
			expectedErr: "result[0] is an error, which is unsupported",
		},
		{
			name:        "incorrect order",
			input:       func(api.Module, context.Context) error { return nil },
			expectedErr: "invalid signature: api.Module parameter must be preceded by context.Context",
		},
		{
			name:        "multiple context.Context",
			input:       func(context.Context, uint64, context.Context) error { return nil },
			expectedErr: "param[2] is a context.Context, which may be defined only once as param[0]",
		},
		{
			name:        "multiple wasm.Module",
			input:       func(context.Context, api.Module, uint64, api.Module) error { return nil },
			expectedErr: "param[3] is a api.Module, which may be defined only once as param[0]",
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parseGoReflectFunc(tc.input)
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

// stack simulates the value stack in a way easy to be tested.
type stack struct {
	vals []uint64
}

func (s *stack) pop() (result uint64) {
	stackTopIndex := len(s.vals) - 1
	result = s.vals[stackTopIndex]
	s.vals = s.vals[:stackTopIndex]
	return
}

func TestPopValues(t *testing.T) {
	stackVals := []uint64{1, 2, 3, 4, 5, 6, 7}
	var tests = []struct {
		name     string
		count    int
		expected []uint64
	}{
		{
			name: "pop zero doesn't allocate a slice ",
		},
		{
			name:     "pop 1",
			count:    1,
			expected: []uint64{7},
		},
		{
			name:     "pop 2",
			count:    2,
			expected: []uint64{6, 7},
		},
		{
			name:     "pop 3",
			count:    3,
			expected: []uint64{5, 6, 7},
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			vals := PopValues(tc.count, (&stack{stackVals}).pop)
			require.Equal(t, tc.expected, vals)
		})
	}
}

func Test_callGoFunc(t *testing.T) {
	tPtr := uintptr(unsafe.Pointer(t))
	callCtx := &CallContext{}

	var tests = []struct {
		name                         string
		input                        interface{}
		inputParams, expectedResults []uint64
	}{
		{
			name:  "() -> ()",
			input: func() {},
		},
		{
			name: "(ctx) -> ()",
			input: func(ctx context.Context) {
				require.Equal(t, testCtx, ctx)
			},
		},
		{
			name: "(ctx, mod) -> ()",
			input: func(ctx context.Context, m api.Module) {
				require.Equal(t, testCtx, ctx)
				require.Equal(t, callCtx, m)
			},
		},
		{
			name: "all supported params and i32 result",
			input: func(v uintptr, w uint32, x uint64, y float32, z float64) uint32 {
				require.Equal(t, tPtr, v)
				require.Equal(t, uint32(math.MaxUint32), w)
				require.Equal(t, uint64(math.MaxUint64), x)
				require.Equal(t, float32(math.MaxFloat32), y)
				require.Equal(t, math.MaxFloat64, z)
				return 100
			},
			inputParams: []uint64{
				api.EncodeExternref(tPtr),
				math.MaxUint32,
				math.MaxUint64,
				api.EncodeF32(math.MaxFloat32),
				api.EncodeF64(math.MaxFloat64),
			},
			expectedResults: []uint64{100},
		},
		{
			name: "all supported params and i32 result - (ctx)",
			input: func(ctx context.Context, v uintptr, w uint32, x uint64, y float32, z float64) uint32 {
				require.Equal(t, testCtx, ctx)
				require.Equal(t, tPtr, v)
				require.Equal(t, uint32(math.MaxUint32), w)
				require.Equal(t, uint64(math.MaxUint64), x)
				require.Equal(t, float32(math.MaxFloat32), y)
				require.Equal(t, math.MaxFloat64, z)
				return 100
			},
			inputParams: []uint64{
				api.EncodeExternref(tPtr),
				math.MaxUint32,
				math.MaxUint64,
				api.EncodeF32(math.MaxFloat32),
				api.EncodeF64(math.MaxFloat64),
			},
			expectedResults: []uint64{100},
		},
		{
			name: "all supported params and i32 result - (ctx, mod)",
			input: func(ctx context.Context, m api.Module, v uintptr, w uint32, x uint64, y float32, z float64) uint32 {
				require.Equal(t, testCtx, ctx)
				require.Equal(t, callCtx, m)
				require.Equal(t, tPtr, v)
				require.Equal(t, uint32(math.MaxUint32), w)
				require.Equal(t, uint64(math.MaxUint64), x)
				require.Equal(t, float32(math.MaxFloat32), y)
				require.Equal(t, math.MaxFloat64, z)
				return 100
			},
			inputParams: []uint64{
				api.EncodeExternref(tPtr),
				math.MaxUint32,
				math.MaxUint64,
				api.EncodeF32(math.MaxFloat32),
				api.EncodeF64(math.MaxFloat64),
			},
			expectedResults: []uint64{100},
		},
	}
	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			_, _, code, err := parseGoReflectFunc(tc.input)
			require.NoError(t, err)

			var results []uint64
			switch code.GoFunc.(type) {
			case api.GoFunction:
				results = code.GoFunc.(api.GoFunction).Call(testCtx, tc.inputParams)
			case api.GoModuleFunction:
				results = code.GoFunc.(api.GoModuleFunction).Call(testCtx, callCtx, tc.inputParams)
			default:
				t.Fatal("unexpected type.")
			}
			require.Equal(t, tc.expectedResults, results)
		})
	}
}
