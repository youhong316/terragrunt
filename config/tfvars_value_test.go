package config

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestParseTfVarsValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		value    string
		expected TfVarsValue
	} {
		{"empty", `""`, tfVars()},
		{"string", `"foo"`, tfVars(str("foo"))},
		{"whitespace string", `"      "`, tfVars(str("      "))},
		{"int", `3`, tfVars(integer(3))},
		{"float", `3.14159`, tfVars(float(3.14159))},
		{"bool", `true`, tfVars(boolean(true))},
		{"interpolation", `"${foo()}"`, tfVars(interp("foo"))},
		{"string interpolation", `"foo${bar()}"`, tfVars(str("foo"), interp("bar"))},
		{"string interpolation string", `"foo${bar()}baz"`, tfVars(str("foo"), interp("bar"), str("baz"))},
		{"string whitespace interpolation string whitespace", `"foo   ${bar()}baz   "`, tfVars(str("foo   "), interp("bar"), str("baz   "))},
		{"string interpolation string interpolation", `"foo${bar()}baz${blah()}"`, tfVars(str("foo"), interp("bar"), str("baz"), interp("blah"))},
		{"string interpolation string interpolation string", `"foo${bar()}baz${blah()}abc"`, tfVars(str("foo"), interp("bar"), str("baz"), interp("blah"), str("abc"))},
		{"interpolation with one string arg", `"${foo("bar")}"`, tfVars(interp("foo", tfVars(str("bar"))))},
		{"interpolation with one int arg", `"${foo(42)}"`, tfVars(interp("foo", tfVars(integer(42))))},
		{"interpolation with one float arg", `"${foo(-42.0)}"`, tfVars(interp("foo", tfVars(float(-42.0))))},
		{"interpolation with one bool arg", `"${foo(false)}"`, tfVars(interp("foo", tfVars(boolean(false))))},
		{"interpolation with multiple string args", `"${foo("bar", "baz", "blah")}"`, tfVars(interp("foo", tfVars(str("bar")), tfVars(str("baz")), tfVars(str("blah"))))},
		{"interpolation with multiple arg types", `"${foo("bar", 99999, 0.333333333, true)}"`, tfVars(interp("foo", tfVars(str("bar")), tfVars(integer(99999)), tfVars(float(0.333333333)), tfVars(boolean(true))))},
		{"interpolation with one interpolated arg", `"${foo("${bar()}")}"`, tfVars(interp("foo", tfVars(interp("bar"))))},
		{"interpolation with one interpolated and string arg", `"${foo("abc${bar()}def")}"`, tfVars(interp("foo", tfVars(str("abc"), interp("bar"), str("def"))))},
		{"interpolation with one interpolated arg with its own string arg", `"${foo("${bar("baz")}")}"`, tfVars(interp("foo", tfVars(interp("bar", tfVars(str("baz"))))))},
		{"interpolation with interpolated args and literal args", `"${foo("${bar()}", -33, true, "hi")}"`, tfVars(interp("foo", tfVars(interp("bar")), tfVars(integer(-33)), tfVars(boolean(true)), tfVars(str("hi"))))},
		{"string interpolation with interpolated args and literal args string", `"abc${foo("${bar()}", -33, true, "hi")}def"`, tfVars(str("abc"), interp("foo", tfVars(interp("bar")), tfVars(integer(-33)), tfVars(boolean(true)), tfVars(str("hi"))), str("def"))},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := ParseTfVarsValue("test", testCase.value)
			if assert.NoError(t, err) && assert.NotNil(t, actual) {
				assert.Equal(t, testCase.expected, *actual)
			}
		})
	}
}

func tfVars(parts ... TfVarsValuePart) TfVarsValue {
	if parts == nil {
		parts = []TfVarsValuePart{}
	}
	return TfVarsValue{Parts: parts}
}

func str(contents string) String {
	return String{Contents: contents}
}

func integer(val int) Integer {
	return Integer{Value: val}
}

func float(val float64) Float {
	return Float{Value: val}
}

func boolean(val bool) Boolean {
	return Boolean{Value: val}
}

func interp(functionName string, args ... TfVarsValue) Interpolation {
	if args == nil {
		args = []TfVarsValue{}
	}

	return Interpolation{FunctionName: functionName, Args: args}
}