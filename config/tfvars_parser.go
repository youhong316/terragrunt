//go:generate pigeon -o tfvars_value.go tfvars_value.peg
//
// The comment above can be used with go generate to automatically compile the PEG grammar in tfvars_value.peg into
// a Go parser. To have the command above take effect, before running go build, you simply run:
//
// go generate ./...
//
package config

import (
	"github.com/gruntwork-io/terragrunt/errors"
	"reflect"
	"fmt"
)

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
	case TfVarsInterpolation:
		return NewTfVarsValue(v), nil
	case []interface{}:
		parts := []TfVarsValuePart{}
		for _, item := range v {
			collapsed, err := wrapTfVarsValue(item)
			if err != nil {
				return TfVarsValue{}, err
			}
			parts = append(parts, []TfVarsValuePart(collapsed)...)
		}
		return TfVarsValue(combineStrings(parts)), nil
	default:
		return TfVarsValue{}, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "string, int, float64, or TfVarsInterpolation", ActualType: reflect.TypeOf(val), Value: val})
	}
}

func combineStrings(parts []TfVarsValuePart) []TfVarsValuePart {
	combinedParts := []TfVarsValuePart{}

	for _, part := range parts {
		if partAsString, ok := part.(TfVarsString); ok && len(combinedParts) > 0 {
			prevPart := combinedParts[len(combinedParts) - 1]
			if prevPartAsString, ok := prevPart.(TfVarsString); ok {
				combinedParts = combinedParts[:len(combinedParts) - 1]
				mergedPart := TfVarsString(string(prevPartAsString) + string(partAsString))
				combinedParts = append(combinedParts, mergedPart)
				continue
			}
		}

		combinedParts = append(combinedParts, part)
	}

	return combinedParts
}

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

type TfVarsValue []TfVarsValuePart

func NewTfVarsValue(parts ... TfVarsValuePart) TfVarsValue {
	return TfVarsValue(parts)
}

type TfVarsValuePart interface {
	Render() string
}

type TfVarsString string

func (val TfVarsString) Render() string {
	return fmt.Sprintf("TfVarsString(%s)", string(val))
}

func (val TfVarsString) String() string {
	return val.Render()
}

type TfVarsInt int

func (val TfVarsInt) Render() string {
	return fmt.Sprintf("TfVarsInt(%d)", int(val))
}

func (val TfVarsInt) String() string {
	return val.Render()
}

type TfVarsFloat float64

func (val TfVarsFloat) Render() string {
	return fmt.Sprintf("TfVarsFloat(%f)", float64(val))
}

func (val TfVarsFloat) String() string {
	return val.Render()
}

type TfVarsBool bool

func (val TfVarsBool) Render() string {
	return fmt.Sprintf("TfVarsBool(%s)", bool(val))
}

func (val TfVarsBool) String() string {
	return val.Render()
}

type TfVarsArray []TfVarsValue

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

func (val TfVarsArray) Render() string {
	return fmt.Sprintf("TfVarsArray(%v)", []TfVarsValue(val))
}

func (val TfVarsArray) String() string {
	return val.Render()
}

type TfVarsInterpolation struct {
	FunctionName string
	Args         []TfVarsValue
}

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

func (val TfVarsInterpolation) Render() string {
	return fmt.Sprintf("TfVarsInterpolation(Name: %s, Args: %v)", val.FunctionName, val.Args)
}

func (val TfVarsInterpolation) String() string {
	return val.Render()
}

func toIfaceSlice(value interface{}) ([]interface{}, error) {
	slice, isSlice := value.([]interface{})
	if !isSlice {
		return nil, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "[]interface{}", ActualType: reflect.TypeOf(value), Value: value})
	}
	return slice, nil
}

type InvalidInterpolation struct {
	ExpectedSyntax string
	ActualSyntax   string
}

func (err InvalidInterpolation) Error() string {
	return fmt.Sprintf("Expected an interpolation body of the format '%s' but got '%s'", err.ExpectedSyntax, err.ActualSyntax)
}