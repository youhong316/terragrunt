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
		return NewTfVarsValue(String{Contents: v}), nil
	case int:
		return NewTfVarsValue(Integer{Value: v}), nil
	case float64:
		return NewTfVarsValue(Float{Value: v}), nil
	case bool:
		return NewTfVarsValue(Boolean{Value: v}), nil
	case Interpolation:
		return NewTfVarsValue(v), nil
	case []interface{}:
		parts := []TfVarsValuePart{}
		for _, item := range v {
			collapsed, err := wrapTfVarsValue(item)
			if err != nil {
				return TfVarsValue{}, err
			}
			parts = append(parts, collapsed.Parts...)
		}
		return NewTfVarsValue(combineStrings(parts)...), nil
	default:
		return TfVarsValue{}, errors.WithStackTrace(UnexpectedParserReturnType{ExpectedType: "string, int, float64, or Interpolation", ActualType: reflect.TypeOf(val), Value: val})
	}
}

func combineStrings(parts []TfVarsValuePart) []TfVarsValuePart {
	combinedParts := []TfVarsValuePart{}

	for _, part := range parts {
		if partAsString, ok := part.(String); ok && len(combinedParts) > 0 {
			prevPart := combinedParts[len(combinedParts) - 1]
			if prevPartAsString, ok := prevPart.(String); ok {
				combinedParts = combinedParts[:len(combinedParts) - 1]
				mergedPart := String{Contents: prevPartAsString.Contents + partAsString.Contents}
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

type TfVarsValue struct {
	Parts []TfVarsValuePart
}

func NewTfVarsValue(parts ... TfVarsValuePart) TfVarsValue {
	if parts == nil {
		parts = []TfVarsValuePart{}
	}
	return TfVarsValue{Parts: parts}
}

type TfVarsValuePart interface {
	Render() string
}

type String struct {
	Contents string
}

func (val String) Render() string {
	return fmt.Sprintf("String{Contents: '%s'}", val.Contents)
}

func (val String) String() string {
	return val.Render()
}

type Integer struct {
	Value int
}

func (val Integer) Render() string {
	return fmt.Sprintf("Integer{Value: %d}", val.Value)
}

func (val Integer) String() string {
	return val.Render()
}

type Float struct {
	Value float64
}

func (val Float) Render() string {
	return fmt.Sprintf("Float{Value: %f}", val.Value)
}

func (val Float) String() string {
	return val.Render()
}

type Boolean struct {
	Value bool
}

func (val Boolean) Render() string {
	return fmt.Sprintf("Boolean{Value: %s}", val.Value)
}

func (val Boolean) String() string {
	return val.Render()
}

type Interpolation struct {
	FunctionName string
	Args         []TfVarsValue
}

func NewInterpolation(name interface{}, args interface{}) (Interpolation, error) {
	argsSlice, err := toIfaceSlice(args)
	if err != nil {
		return Interpolation{}, err
	}

	parsedArgs := []TfVarsValue{}
	for _, arg := range argsSlice {
		parsedArg, err := wrapTfVarsValue(arg)
		if err != nil {
			return Interpolation{}, err
		}
		parsedArgs = append(parsedArgs, parsedArg)
	}

	return Interpolation{FunctionName: fmt.Sprintf("%v", name), Args: parsedArgs}, nil
}

func (val Interpolation) Render() string {
	return fmt.Sprintf("Interpolation{Name: %s, Args: %v}", val.FunctionName, val.Args)
}

func (val Interpolation) String() string {
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