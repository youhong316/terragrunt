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
		{"empty string", `""`, str()},
		{"string", `"foo"`, str(chars("foo"))},
		{"string with curly braces", `"{foo}"`, str(chars("{foo}"))},
		{"string with dollar sign", `"$foo"`, str(chars("$foo"))},
		{"string with escapes", `"\"foo\""`, str(chars(`"foo"`))},
		{"whitespace string", `"      "`, str(chars("      "))},
		{"int", `3`, integer(3)},
		{"float", `3.14159`, float(3.14159)},
		{"bool", `true`, boolean(true)},
		{"empty array", `[]`, array()},
		{"string array", `["foo", "bar", "baz"]`, array(str(chars("foo")), str(chars("bar")), str(chars("baz")))},
		{"int array", `[1, 2, 3]`, array(integer(1), integer(2), integer(3))},
		{"array with maps", `[{}, {foo = "bar"}]`, array(tfVarsMap(), tfVarsMap(keyValue(str(chars("foo")), str(chars("bar")))))},
		{"mixed types array", `["foo", 2, true]`, array(str(chars("foo")), integer(2), boolean(true))},
		{"array without commas", `["foo" 2 true]`, array(str(chars("foo")), integer(2), boolean(true))},
		{"array whitespace", `[    1,2     ,         3]`, array( integer(1), integer(2), integer(3))},
		{"nested array", `[["foo"]]`, array(array(str(chars("foo"))))},
		{"nested arrays", `[["foo"], ["bar"], [1, 2, 3]]`, array(array(str(chars("foo"))), array(str(chars("bar"))), array(integer(1), integer(2), integer(3)))},
		{"array with interpolation", `["${foo()}"]`, array(str(interp("foo")))},
		{"empty map", `{}`, tfVarsMap()},
		{"map with string key string value", `{foo = "bar"}`, tfVarsMap(keyValue(str(chars("foo")), str(chars("bar"))))},
		{"map with string key int value", `{foo = 5}`, tfVarsMap(keyValue(str(chars("foo")), integer(5)))},
		{"map with string key bool value", `{foo = true}`, tfVarsMap(keyValue(str(chars("foo")), boolean(true)))},
		{"map with string key array value", `{foo = [1, 2, 3]}`, tfVarsMap(keyValue(str(chars("foo")), array(integer(1), integer(2), integer(3))))},
		{"map with string key map value", `{foo = {bar = "baz"}}`, tfVarsMap(keyValue(str(chars("foo")), tfVarsMap(keyValue(str(chars("bar")), str(chars("baz"))))))},
		{"map with multiple keys and values", `{foo = "bar", baz = 1.0, blah = true}`, tfVarsMap(keyValue(str(chars("foo")), str(chars("bar"))), keyValue(str(chars("baz")), float(1.0)), keyValue(str(chars("blah")), boolean(true)))},
		{"map with multiple keys and values and no commas", `{foo = "bar" baz = 1.0 blah = true}`, tfVarsMap(keyValue(str(chars("foo")), str(chars("bar"))), keyValue(str(chars("baz")), float(1.0)), keyValue(str(chars("blah")), boolean(true)))},
		{"map with interpolated value", `{foo = "${bar()}"}`, tfVarsMap(keyValue(str(chars("foo")), str(interp("bar"))))},
		{"interpolation", `"${foo()}"`, str(interp("foo"))},
		{"escaped interpolation", `"$${foo()}"`, str(chars("$${foo()}"))},
		{"string interpolation", `"foo${bar()}"`, str(chars("foo"), interp("bar"))},
		{"string interpolation string", `"foo${bar()}baz"`, str(chars("foo"), interp("bar"), chars("baz"))},
		{"string whitespace interpolation string whitespace", `"foo   ${bar()}baz   "`, str(chars("foo   "), interp("bar"), chars("baz   "))},
		{"string interpolation string interpolation", `"foo${bar()}baz${blah()}"`, str(chars("foo"), interp("bar"), chars("baz"), interp("blah"))},
		{"string interpolation string interpolation string", `"foo${bar()}baz${blah()}abc"`, str(chars("foo"), interp("bar"), chars("baz"), interp("blah"), chars("abc"))},
		{"interpolation with one string arg", `"${foo("bar")}"`, str(interp("foo", str(chars("bar"))))},
		{"interpolation with one int arg", `"${foo(42)}"`, str(interp("foo", integer(42)))},
		{"interpolation with one float arg", `"${foo(-42.0)}"`, str(interp("foo", float(-42.0)))},
		{"interpolation with one bool arg", `"${foo(false)}"`, str(interp("foo", boolean(false)))},
		{"interpolation with one array arg", `"${foo(["foo", "bar", "baz"])}"`, str(interp("foo", array(str(chars("foo")), str(chars("bar")), str(chars("baz")))))},
		{"interpolation with multiple string args", `"${foo("bar", "baz", "blah")}"`, str(interp("foo", str(chars("bar")), str(chars("baz")), str(chars("blah"))))},
		{"interpolation with multiple arg types", `"${foo("bar", 99999, 0.333333333, true, [42.0])}"`, str(interp("foo", str(chars("bar")), integer(99999), float(0.333333333), boolean(true), array(float(42.0))))},
		{"interpolation with one interpolated arg", `"${foo("${bar()}")}"`, str(interp("foo", str(interp("bar"))))},
		{"interpolation with one interpolated and string arg", `"${foo("abc${bar()}def")}"`, str(interp("foo", str(chars("abc"), interp("bar"), chars("def"))))},
		{"interpolation with one interpolated arg with its own string arg", `"${foo("${bar("baz")}")}"`, str(interp("foo", str(interp("bar", str(chars("baz"))))))},
		{"interpolation with interpolated args and literal args", `"${foo("${bar()}", -33, true, "hi", {foo = "bar"})}"`, str(interp("foo", str(interp("bar")), integer(-33), boolean(true), str(chars("hi")), tfVarsMap(keyValue(str(chars("foo")), str(chars("bar"))))))},
		{"string interpolation with interpolated args and literal args string", `"abc${foo("${bar([true, true, true])}", -33, true, "hi")}def"`, str(chars("abc"), interp("foo", str(interp("bar", array(boolean(true), boolean(true), boolean(true)))), integer(-33), boolean(true), str(chars("hi"))), chars("def"))},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := ParseTfVarsValue("test", testCase.value)
			if assert.NoError(t, err) {
				assert.Equal(t, testCase.expected, actual)
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

func chars(contents string) TfVarsChars {
	return TfVarsChars(contents)
}

func str(parts ... TfVarsValue) TfVarsString {
	if parts == nil {
		parts = []TfVarsValue{}
	}
	return TfVarsString(parts)
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

func array(items ...TfVarsValue) TfVarsArray {
	if items == nil {
		items = []TfVarsValue{}
	}
	return TfVarsArray(items)
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
