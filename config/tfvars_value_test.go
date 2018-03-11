package config

import (
	"github.com/gruntwork-io/terragrunt/errors"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestParseTfVarsValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		value    string
		expected TfVarsValue
	}{
		{"empty string", `""`, tfVars()},
		{"string", `"foo"`, tfVars(str("foo"))},
		{"string with curly braces", `"{foo}"`, tfVars(str("{foo}"))},
		{"string with dollar sign", `"$foo"`, tfVars(str("$foo"))},
		{"string with escapes", `"\"foo\""`, tfVars(str(`"foo"`))},
		{"whitespace string", `"      "`, tfVars(str("      "))},
		{"int", `3`, tfVars(integer(3))},
		{"float", `3.14159`, tfVars(float(3.14159))},
		{"bool", `true`, tfVars(boolean(true))},
		{"empty array", `[]`, tfVars(array(t))},
		{"string array", `["foo", "bar", "baz"]`, tfVars(array(t, "foo", "bar", "baz"))},
		{"int array", `[1, 2, 3]`, tfVars(array(t, 1, 2, 3))},
		{"array with maps", `[{}, {foo = "bar"}]`, tfVars(array(t, tfVarsMap(), tfVarsMap(keyValue(tfVars(str("foo")), tfVars(str("bar"))))))},
		{"mixed types array", `["foo", 2, true]`, tfVars(array(t, "foo", 2, true))},
		{"array whitespace", `[    1,2     ,         3]`, tfVars(array(t, 1, 2, 3))},
		{"nested array", `[["foo"]]`, tfVars(array(t, array(t, "foo")))},
		{"nested arrays", `[["foo"], ["bar"], [1, 2, 3]]`, tfVars(array(t, array(t, "foo"), array(t, "bar"), array(t, 1, 2, 3)))},
		{"array with interpolation", `["${foo()}"]`, tfVars(array(t, interp("foo")))},
		{"empty map", `{}`, tfVars(tfVarsMap())},
		{"map with string key string value", `{foo = "bar"}`, tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(str("bar")))))},
		{"map with string key int value", `{foo = 5}`, tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(integer(5)))))},
		{"map with string key bool value", `{foo = true}`, tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(boolean(true)))))},
		{"map with string key array value", `{foo = [1, 2, 3]}`, tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(array(t, 1, 2, 3)))))},
		{"map with string key map value", `{foo = {bar = "baz"}}`, tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(tfVarsMap(keyValue(tfVars(str("bar")), tfVars(str("baz"))))))))},
		{"map with multiple keys and values", `{foo = "bar", baz = 1.0, blah = true}`, tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(str("bar"))), keyValue(tfVars(str("baz")), tfVars(float(1.0))), keyValue(tfVars(str("blah")), tfVars(boolean(true)))))},
		{"map with interpolated value", `{foo = "${bar()}"}`, tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(interp("bar")))))},
		{"interpolation", `"${foo()}"`, tfVars(interp("foo"))},
		{"escaped interpolation", `"$${foo()}"`, tfVars(str("$${foo()}"))},
		{"string interpolation", `"foo${bar()}"`, tfVars(str("foo"), interp("bar"))},
		{"string interpolation string", `"foo${bar()}baz"`, tfVars(str("foo"), interp("bar"), str("baz"))},
		{"string whitespace interpolation string whitespace", `"foo   ${bar()}baz   "`, tfVars(str("foo   "), interp("bar"), str("baz   "))},
		{"string interpolation string interpolation", `"foo${bar()}baz${blah()}"`, tfVars(str("foo"), interp("bar"), str("baz"), interp("blah"))},
		{"string interpolation string interpolation string", `"foo${bar()}baz${blah()}abc"`, tfVars(str("foo"), interp("bar"), str("baz"), interp("blah"), str("abc"))},
		{"interpolation with one string arg", `"${foo("bar")}"`, tfVars(interp("foo", tfVars(str("bar"))))},
		{"interpolation with one int arg", `"${foo(42)}"`, tfVars(interp("foo", tfVars(integer(42))))},
		{"interpolation with one float arg", `"${foo(-42.0)}"`, tfVars(interp("foo", tfVars(float(-42.0))))},
		{"interpolation with one bool arg", `"${foo(false)}"`, tfVars(interp("foo", tfVars(boolean(false))))},
		{"interpolation with one array arg", `"${foo(["foo", "bar", "baz"])}"`, tfVars(interp("foo", tfVars(array(t, "foo", "bar", "baz"))))},
		{"interpolation with multiple string args", `"${foo("bar", "baz", "blah")}"`, tfVars(interp("foo", tfVars(str("bar")), tfVars(str("baz")), tfVars(str("blah"))))},
		{"interpolation with multiple arg types", `"${foo("bar", 99999, 0.333333333, true, [42.0])}"`, tfVars(interp("foo", tfVars(str("bar")), tfVars(integer(99999)), tfVars(float(0.333333333)), tfVars(boolean(true)), tfVars(array(t, 42.0))))},
		{"interpolation with one interpolated arg", `"${foo("${bar()}")}"`, tfVars(interp("foo", tfVars(interp("bar"))))},
		{"interpolation with one interpolated and string arg", `"${foo("abc${bar()}def")}"`, tfVars(interp("foo", tfVars(str("abc"), interp("bar"), str("def"))))},
		{"interpolation with one interpolated arg with its own string arg", `"${foo("${bar("baz")}")}"`, tfVars(interp("foo", tfVars(interp("bar", tfVars(str("baz"))))))},
		{"interpolation with interpolated args and literal args", `"${foo("${bar()}", -33, true, "hi", {foo = "bar"})}"`, tfVars(interp("foo", tfVars(interp("bar")), tfVars(integer(-33)), tfVars(boolean(true)), tfVars(str("hi")), tfVars(tfVarsMap(keyValue(tfVars(str("foo")), tfVars(str("bar")))))))},
		{"string interpolation with interpolated args and literal args string", `"abc${foo("${bar([true, true, true])}", -33, true, "hi")}def"`, tfVars(str("abc"), interp("foo", tfVars(interp("bar", tfVars(array(t, true, true, true)))), tfVars(integer(-33)), tfVars(boolean(true)), tfVars(str("hi"))), str("def"))},
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

func TestParseTfVarsValueErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		value    string
		expected error
	}{
		{"empty", ``, &parserError{}},
		{"naked value", `foo`, &parserError{}},
		{"missing closing quote", `"foo`, &parserError{}},
		{"missing opening quote", `foo"`, &parserError{}},
		{"extra quote", `"foo""`, &parserError{}},
		{"invalid number", `3.4.3`, &parserError{}},
		{"missing closing curly brace", `"${foo()"`, InvalidInterpolation{}},
		{"not a function call", `"${foo}"`, InvalidInterpolation{}},
		{"missing closing bracket", `[1, 2, 3`, &parserError{}},
		{"missing opening bracket", `1, 2, 3]`, &parserError{}},
		{"missing comma", `[1 2 3]`, &parserError{}},
		{"missing double quotes", `[foo]`, &parserError{}},
		{"missing closing curly brace", `{foo = "bar"`, &parserError{}},
		{"missing opening curly brace", `foo = "bar"}`, &parserError{}},
		{"missing equals", `{foo "bar"}`, &parserError{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := ParseTfVarsValue("test", testCase.value)
			if assert.Error(t, err, "Expected error, but got nil. Parsed value: %v", actual) {
				unwrapped := unwrapParserError(t, err, testCase.expected)
				assert.IsType(t, testCase.expected, unwrapped, "Actual error message: %v", unwrapped)
			}
		})
	}
}

// The parser always returns a wrapped list of parserErrors. Unwrap to the first of these.
func unwrapParserError(t *testing.T, actualErr error, expectedErr error) error {
	unwrapped := errors.Unwrap(actualErr)
	list, isList := unwrapped.(errList)

	if !isList || len(list) == 0 {
		t.Fatalf("Expected error to be a non-empty errList, but got a type %v with contents %v:", reflect.TypeOf(actualErr), actualErr)
	}

	firstErr := list[0]
	asParserErr, isParserErr := firstErr.(*parserError)
	if !isParserErr {
		t.Fatalf("Expected first error to be a parserError but got an error of type %v: %v", reflect.TypeOf(firstErr), firstErr)
	}

	// If we are expecting a custom error type, then we need to pull it out of the parserError
	if reflect.TypeOf(expectedErr) != reflect.TypeOf(&parserError{}) {
		return asParserErr.Inner
	}

	return asParserErr
}

func tfVars(parts ...TfVarsValuePart) TfVarsValue {
	if parts == nil {
		parts = []TfVarsValuePart{}
	}
	return TfVarsValue(parts)
}

func str(contents string) TfVarsString {
	return TfVarsString(contents)
}

func integer(val int) TfVarsInt {
	return TfVarsInt(val)
}

func float(val float64) TfVarsFloat {
	return TfVarsFloat(val)
}

func boolean(val bool) TfVarsBool {
	return TfVarsBool(val)
}

func array(t *testing.T, items ...interface{}) TfVarsArray {
	parts := []TfVarsValue{}

	for _, item := range items {
		wrapped, err := wrapTfVarsValue(item)
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, wrapped)
	}

	return TfVarsArray(parts)
}

func tfVarsMap(keyValuePairs ...TfVarsKeyValue) TfVarsMap {
	if keyValuePairs == nil {
		keyValuePairs = []TfVarsKeyValue{}
	}
	return TfVarsMap(keyValuePairs)
}

func keyValue(key TfVarsValue, value TfVarsValue) TfVarsKeyValue {
	return TfVarsKeyValue{Key: key, Value: value}
}

func interp(functionName string, args ...TfVarsValue) TfVarsInterpolation {
	if args == nil {
		args = []TfVarsValue{}
	}

	return TfVarsInterpolation{FunctionName: functionName, Args: args}
}
