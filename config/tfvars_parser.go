//go:generate pigeon -o tfvars_value.go tfvars_value.peg
//
// The comment above can be used with go generate to automatically compile the PEG grammar in tfvars_value.peg into
// a Go parser. To have the command above take effect, before running go build, you simply run:
//
// go generate $(glide novendor)
//
package config

import (
	"fmt"
	"github.com/gruntwork-io/terragrunt/errors"
	"reflect"
)

// Parse a value from a Terraform .tfvars file. For example, if the .tfvars file contains:
//
// foo = "bar"
//
// This method can be used to parse "bar" into a TfVarsValue, which is an abstract syntax tree (AST). The reason we
// have this method rather than using the official HCL parser is that Terragrunt supports interpolation functions in
// .tfvars files such as:
//
// foo = "${some_function()}"
//
// The parsing and processing of interpolation functions is only available in Terraform and not HCL itself, so we have
// created our own parser for them here.
func ParseTfVarsValue(filename string, value string) (*TfVarsValue, error) {
	out, err := Parse(filename, []byte(value))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	tfVarsValue, ok := out.(TfVarsValue)
	if !ok {
		return nil, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "[]TfVarsValuePart", ActualType: reflect.TypeOf(out), Value: out})
	}

	return &tfVarsValue, nil
}

// Take a value returned by the PEG parser and depending on its type, wrap it with an appropriate TfVarsValue.
func wrapTfVarsValue(val interface{}) (TfVarsValue, error) {
	switch v := val.(type) {
	case string:
		return NewTfVarsValue(TfVarsString(v)), nil
	case int:
		return NewTfVarsValue(TfVarsInt(v)), nil
	case float64:
		return NewTfVarsValue(TfVarsFloat(v)), nil
	case bool:
		return NewTfVarsValue(TfVarsBool(v)), nil
	case TfVarsArray:
		return NewTfVarsValue(v), nil
	case TfVarsMap:
		return NewTfVarsValue(v), nil
	case TfVarsInterpolation:
		return NewTfVarsValue(v), nil
	case []interface{}:
		return wrapTfVarsSlice(v)
	default:
		return TfVarsValue{}, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "string, int, float64, bool, TfVarsArray, TfVarsMap, TfVarsInterpolation, or []interface{}", ActualType: reflect.TypeOf(val), Value: val})
	}
}

// Take a slice returned by the PEG parser and, depending on its type, wrap it with an appropriate TfVarsValue. If the
// slice contains strings, this method will combine adjacent strings down into a single string.
func wrapTfVarsSlice(slice []interface{}) (TfVarsValue, error) {
	parts := []TfVarsValuePart{}
	for _, item := range slice {
		collapsed, err := wrapTfVarsValue(item)
		if err != nil {
			return TfVarsValue{}, err
		}
		parts = append(parts, []TfVarsValuePart(collapsed)...)
	}
	return TfVarsValue(combineStrings(parts)), nil
}

// Go through the list of TfVarsValueParts and combine adjacent TfVarsStrings into a single TfVarsString. The reason we
// do this is that the PEG parser reads text one character at a time, so it may return a separate TfVarsString for
// each character of a string, which is quite inconvenient for processing. This method combines those strings into one.
func combineStrings(parts []TfVarsValuePart) []TfVarsValuePart {
	combinedParts := []TfVarsValuePart{}

	for _, currPart := range parts {
		if currPartAsString, currPartIsString := currPart.(TfVarsString); currPartIsString && len(combinedParts) > 0 {
			prevPart := combinedParts[len(combinedParts)-1]
			if prevPartAsString, prevPartIsString := prevPart.(TfVarsString); prevPartIsString {
				combinedParts = combinedParts[:len(combinedParts)-1]
				currPart = TfVarsString(string(prevPartAsString) + string(currPartAsString))
			}
		}

		combinedParts = append(combinedParts, currPart)
	}

	return combinedParts
}

// An AST that represents a single value in a .tfvars file. Each value may consists of multiple TfVarsValuePart parts.
type TfVarsValue []TfVarsValuePart

// Create a new TfVarsValue from the given parts
func NewTfVarsValue(parts ...TfVarsValuePart) TfVarsValue {
	return TfVarsValue(parts)
}

// Represents one part of a value in a .tfvars file, such as a string, int, or bool.
type TfVarsValuePart interface {
	// Go is a shitty language, so a struct only implements an interface if it implements all the methods from that
	// interface. We don't actually need any methods in TfVarsValuePart, so we have to have this useless marker method
	// here so subtypes have something to implement. For more info, see: https://golang.org/doc/faq#guarantee_satisfies_interface
	ImplementsTfVarsValue()
}

// A wrapper type for a string in a .tfvars file. E.g.
//
// foo = "bar"
//
type TfVarsString string

// Implement the Go Stringer interface
func (val TfVarsString) String() string {
	return fmt.Sprintf("TfVarsString(%s)", string(val))
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsString) ImplementsTfVarsValue() {}


// A wrapper type for an int in a .tfvars file. E.g.
//
// foo = 42
//
type TfVarsInt int

// Implement the Go Stringer interface
func (val TfVarsInt) String() string {
	return fmt.Sprintf("TfVarsInt(%d)", int(val))
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsInt) ImplementsTfVarsValue() {}

// A wrapper type for a float in a .tfvars file. E.g.
//
// foo = 42.0
//
type TfVarsFloat float64

// Implement the Go Stringer interface
func (val TfVarsFloat) String() string {
	return fmt.Sprintf("TfVarsFloat(%f)", float64(val))
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsFloat) ImplementsTfVarsValue() {}

// A wrapper type for a bool in a .tfvars file. E.g.
//
// foo = true
//
type TfVarsBool bool

// Implement the Go Stringer interface
func (val TfVarsBool) String() string {
	return fmt.Sprintf("TfVarsBool(%s)", bool(val))
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsBool) ImplementsTfVarsValue() {}

// A wrapper type for an array in a .tfvars file. E.g.
//
// foo = [1, 2, 3]
//
type TfVarsArray []TfVarsValue

// Create an array from an interface returned by the PEG parser. We expect this interface to actually be a slice of
// interfaces, each of which can be any valid .tfvars value (e.g., string, bool, int, etc).
func NewArray(items interface{}) (TfVarsArray, error) {
	itemsSlice, err := toIfaceSlice(items)
	if err != nil {
		return TfVarsArray{}, err
	}

	wrappedItems := []TfVarsValue{}
	for _, item := range itemsSlice {
		wrapped, err := wrapTfVarsValue(item)
		if err != nil {
			return TfVarsArray{}, err
		}
		wrappedItems = append(wrappedItems, wrapped)
	}

	return TfVarsArray(wrappedItems), nil
}

// Implement the Go Stringer interface
func (val TfVarsArray) String() string {
	return fmt.Sprintf("TfVarsArray(%v)", []TfVarsValue(val))
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsArray) ImplementsTfVarsValue() {}

// A wrapper type for a map in a .tfvars file. E.g.
//
// foo = {
//   bar = "baz"
// }
//
// Note that as the keys could be any arbitrary .tfvars type (e.g., string, interpolation, etc.), and we represent some
// of those types as a list, we cannot simply wrap the native Go map, as that does not allow using a slice for a key.
// Instead, we treat a map as a list of (key, value) pairs.
type TfVarsMap []TfVarsKeyValue

// Go doesn't have tuples, so this is a small wrapper struct the PEG parser can use to wrap a (key, value) pair
type KeyValue struct {
	Key   interface{}
	Value interface{}
}

// Implement the Go Stringer interface
func (keyValue KeyValue) String() string {
	return fmt.Sprintf("KeyValue('%v': '%v')", keyValue.Key, keyValue.Value)
}

// Used to store a (key, value) pair where the key and the value can be a value from a .tfvars file (e.g., string,
// interpolation, int, etc.)
type TfVarsKeyValue struct {
	Key   TfVarsValue
	Value TfVarsValue
}

// Implement the Go Stringer interface
func (keyValue TfVarsKeyValue) String() string {
	return fmt.Sprintf("TfVarsKeyValue('%v': '%v')", keyValue.Key, keyValue.Value)
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsKeyValue) ImplementsTfVarsValue() {}

// Create a map for an interface returned by the PEG parser. We expect the interface to contain a slice of interfaces,
// each of which is actually of type KeyValue.
func NewMap(items interface{}) (TfVarsMap, error) {
	itemsSlice, err := toIfaceSlice(items)
	if err != nil {
		return TfVarsMap{}, err
	}

	wrappedItems := []TfVarsKeyValue{}
	for _, item := range itemsSlice {
		asKeyValue, isKeyValue := item.(KeyValue)
		if !isKeyValue {
			return TfVarsMap{}, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "KeyValue", ActualType: reflect.TypeOf(item), Value: item})
		}

		wrappedKey, err := wrapTfVarsValue(asKeyValue.Key)
		if err != nil {
			return TfVarsMap{}, err
		}

		wrappedValue, err := wrapTfVarsValue(asKeyValue.Value)
		if err != nil {
			return TfVarsMap{}, err
		}

		wrappedItems = append(wrappedItems, TfVarsKeyValue{Key: wrappedKey, Value: wrappedValue})
	}

	return TfVarsMap(wrappedItems), nil
}

// Implement the Go Stringer interface
func (val TfVarsMap) String() string {
	return fmt.Sprintf("TfVarsMap(%v)", []TfVarsKeyValue(val))
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsMap) ImplementsTfVarsValue() {}

// A wrapper type for an interpolation in a .tfvars file. E.g.
//
// foo = "${foo()}"
//
type TfVarsInterpolation struct {
	FunctionName string
	Args         []TfVarsValue
}

// Create a new interpolation for a function name and args returned by the PEG parser. We expect the function name to
// be a string and the args to be a slice of interfaces, where each one is a .tfvars value (e.g., string,
// interpolation, int, etc.).
func NewInterpolation(name interface{}, args interface{}) (TfVarsInterpolation, error) {
	argsSlice, err := toIfaceSlice(args)
	if err != nil {
		return TfVarsInterpolation{}, err
	}

	parsedArgs := []TfVarsValue{}
	for _, arg := range argsSlice {
		parsedArg, err := wrapTfVarsValue(arg)
		if err != nil {
			return TfVarsInterpolation{}, err
		}
		parsedArgs = append(parsedArgs, parsedArg)
	}

	return TfVarsInterpolation{FunctionName: fmt.Sprintf("%v", name), Args: parsedArgs}, nil
}

// Implement the Go Stringer interface
func (val TfVarsInterpolation) String() string {
	return fmt.Sprintf("TfVarsInterpolation(Name: %s, Args: %v)", val.FunctionName, val.Args)
}

// This useless empty method is necessary to label this struct as implementing the TfVarsValuePart interface
func (val TfVarsInterpolation) ImplementsTfVarsValue() {}

// Convert the given interface to a slice of interfaces. The PEG parser mostly returns interface for everything, but
// under the hood, most of the values are actually slices of interfaces, so this is a reusable method for doing the
// type coercion.
func toIfaceSlice(value interface{}) ([]interface{}, error) {
	slice, isSlice := value.([]interface{})
	if !isSlice {
		return nil, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "[]interface{}", ActualType: reflect.TypeOf(value), Value: value})
	}
	return slice, nil
}

// Custom error types

type UnexpectedParserReturnType struct {
	ExpectedType string
	ActualType   reflect.Type
	Value        interface{}
}

func (err UnexpectedParserReturnType) Error() string {
	return fmt.Sprintf("Expected parser to return type %v but got %v. Value: %v", err.ExpectedType, err.ActualType, err.Value)
}

type UnexpectedListLength struct {
	ExpectedLength int
	ActualLength   int
}

func (err UnexpectedListLength) Error() string {
	return fmt.Sprintf("Expected parser to return a list of length %d but got %d", err.ExpectedLength, err.ActualLength)
}

type InvalidInterpolation struct {
	ExpectedSyntax string
	ActualSyntax   string
}

func (err InvalidInterpolation) Error() string {
	return fmt.Sprintf("Expected an interpolation of the format '%s' but got '%s'", err.ExpectedSyntax, err.ActualSyntax)
}
