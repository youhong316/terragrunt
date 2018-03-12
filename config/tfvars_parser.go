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
	"github.com/gruntwork-io/terragrunt/options"
	"strings"
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
func ParseTfVarsValue(filename string, value string) (TfVarsValue, error) {
	out, err := Parse(filename, []byte(value))
	if err != nil {
		return nil, errors.WithStackTrace(err)
	}

	tfVarsValue, ok := out.(TfVarsValue)
	if !ok {
		return nil, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "TfVarsValue", ActualType: reflect.TypeOf(out), Value: out})
	}

	return tfVarsValue, nil
}

// Take a value returned by the PEG parser and depending on its type, wrap it with an appropriate TfVarsValue.
func wrapTfVarsValue(val interface{}) (TfVarsValue, error) {
	switch v := val.(type) {
	case int:
		return TfVarsInt(v), nil
	case float64:
		return TfVarsFloat(v), nil
	case bool:
		return TfVarsBool(v), nil
	case TfVarsString:
		return v, nil
	case TfVarsArray:
		return v, nil
	case TfVarsMap:
		return v, nil
	case string:
		return wrapTfVarsSlice([]interface{}{v})
	case []interface{}:
		return wrapTfVarsSlice(v)
	default:
		return nil, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "int, float64, bool, TfVarsString, TfVarsArray, TfVarsMap, or []interface{}", ActualType: reflect.TypeOf(val), Value: val})
	}
}

// Take a slice returned by the PEG parser, which we should only get for strings, and wrap it as a TfVarsString.
func wrapTfVarsSlice(slice []interface{}) (TfVarsString, error) {
	out := []TfVarsValue{}

	for _, item := range slice {
		switch v := item.(type) {
		case string:
			// The PEG parser parses one character at a time, so we may get a bunch of strings in a row in the slice.
			// Instead of storing each one as a separate TfVarsChars, we use prev to combine adjacent strings together.
			next := TfVarsChars(v)

			if len(out) > 0 {
				prev := out[len(out) - 1]
				prevAsChars, prevIsChars := prev.(TfVarsChars)
				if prevIsChars {
					out = out[:len(out) - 1]
					next = TfVarsChars(string(prevAsChars) + v)
				}
			}

			out = append(out, next)
		case TfVarsInterpolation:
			out = append(out, v)
		default:
			return nil, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "string or TfVarsInterpolation", ActualType: reflect.TypeOf(item), Value: item})
		}
	}

	return TfVarsString(out), nil
}

// An AST that represents a single value in a .tfvars file, such as a string, interpolation, int, etc.
type TfVarsValue interface {
	// Resolve the value. For all "primitive" types such as string, int, etc, this should just return the underlying
	// value. For interpolations, this should execute the interpolation and return the result.
	Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error)
}

// A wrapper type for a string in a .tfvars file. E.g.
//
// foo = "bar"
//
// Note that a string could also contain an interpolation:
//
// foo = "${bar()}"
//
// Or even a mix of string and interpolation:
//
// foo = "abc ${def()} ghi"
//
// Therefore, we represent a string as a list of TfVarsValue parts.
type TfVarsString []TfVarsValue

// Implement the Go Stringer interface
func (val TfVarsString) String() string {
	out := []string{}

	for _, part := range val {
		out = append(out, fmt.Sprintf("%v", part))
	}

	return fmt.Sprintf("TfVarsString(%s)", strings.Join(out, ""))
}

// Implement the TfVarsValue interface
func (val TfVarsString) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	resolved, isResolved, err := val.resolveIfSingleInterpolation(include, terragruntOptions)
	if err != nil {
		return nil, err
	}
	if isResolved {
		return resolved, nil
	}

	out := []string{}

	for _, part := range val {
		resolved, err := part.Resolve(include, terragruntOptions)
		if err != nil {
			return nil, err
		}
		out = append(out, fmt.Sprintf("%v", resolved))
	}

	return strings.Join(out, ""), nil
}

// If this TfVarsString contains a single item which is an interpolation. E.g.,:
//
// foo = "${bar()}"
//
// Then we resolve it immediately, return whatever the underlying interpolation returned, and true; otherwise, we
// return false. We handle this as a special case because the single interpolation sometimes needs to be rendered not
// as a string but as an array or map.
func (val TfVarsString) resolveIfSingleInterpolation(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, bool, error) {
	if len(val) != 1 {
		return nil, false, nil
	}

	asInterpolation, isInterpolation := val[0].(TfVarsInterpolation)
	if !isInterpolation {
		return nil, false, nil
	}

	resolved, err := asInterpolation.Resolve(include, terragruntOptions)
	return resolved, true, err
}

// A wrapper type for a string in a .tfvars file. E.g.
//
// foo = "bar"
//
// Note that unlike TfVarsString, this type can ONLY contain plain characters and no interpolations.
type TfVarsChars string

// Implement the Go Stringer interface
func (val TfVarsChars) String() string {
	return fmt.Sprintf("TfVarsChars(%s)", string(val))
}

// Implement the TfVarsValue interface
func (val TfVarsChars) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	return string(val), nil
}

// A wrapper type for an int in a .tfvars file. E.g.
//
// foo = 42
//
type TfVarsInt int

// Implement the Go Stringer interface
func (val TfVarsInt) String() string {
	return fmt.Sprintf("TfVarsInt(%d)", int(val))
}

// Implement the TfVarsValue interface
func (val TfVarsInt) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	return int(val), nil
}

// A wrapper type for a float in a .tfvars file. E.g.
//
// foo = 42.0
//
type TfVarsFloat float64

// Implement the Go Stringer interface
func (val TfVarsFloat) String() string {
	return fmt.Sprintf("TfVarsFloat(%f)", float64(val))
}

// Implement the TfVarsValue interface
func (val TfVarsFloat) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	return float64(val), nil
}

// A wrapper type for a bool in a .tfvars file. E.g.
//
// foo = true
//
type TfVarsBool bool

// Implement the Go Stringer interface
func (val TfVarsBool) String() string {
	return fmt.Sprintf("TfVarsBool(%s)", bool(val))
}

// Implement the TfVarsValue interface
func (val TfVarsBool) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	return bool(val), nil
}

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

// Implement the TfVarsValue interface
func (val TfVarsArray) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	resolved := []interface{}{}

	for _, item := range val {
		resolvedItem, err := item.Resolve(include, terragruntOptions)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, resolvedItem)
	}

	return resolved, nil
}

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

// Implement the TfVarsValue interface
func (val TfVarsMap) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	resolved := map[interface{}]interface{}{}

	for _, keyValue := range val {
		resolvedKey, err := keyValue.Key.Resolve(include, terragruntOptions)
		if err != nil {
			return nil, err
		}

		resolvedValue, err := keyValue.Value.Resolve(include, terragruntOptions)
		if err != nil {
			return nil, err
		}

		resolved[resolvedKey] = resolvedValue
	}

	return resolved, nil
}

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

// Implement the TfVarsValue interface
func (val TfVarsInterpolation) Resolve(include *IncludeConfig, terragruntOptions *options.TerragruntOptions) (interface{}, error) {
	resolvedArgs := []interface{}{}

	for _, arg := range val.Args {
		resolvedArg, err := arg.Resolve(include, terragruntOptions)
		if err != nil {
			return nil, err
		}
		resolvedArgs = append(resolvedArgs, resolvedArg)
	}

	return executeTerragruntHelperFunction(val.FunctionName, resolvedArgs, include, terragruntOptions)
}

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
