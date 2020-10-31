package compile

import (
	"fmt"
	"strconv"

	"github.com/pdk/meh/lex"
	"github.com/pdk/meh/parser"
)

// Convert a parse tree to an executable function

// Value is a value.
type Value interface{}

// Context is the current name->value map.
type Context map[string]Value

// FlowChangeType indicates the type of flow control change.
type FlowChangeType byte

// Which kinds of flow control changes that are handled.
const (
	None FlowChangeType = iota
	Return
	Break
	Continue
)

// FlowChange is what is returned by an Expr when there is a change of flow.
type FlowChange struct {
	Type FlowChangeType
	Value
}

// flowChange checks if the value is a FlowChange.
func flowChange(v Value) FlowChangeType {
	change, ok := v.(FlowChange)
	if !ok {
		return None
	}
	return change.Type
}

// NewReturn produces a Return FlowChange.
func NewReturn(values ...Value) Value {
	return FlowChange{
		Type:  Return,
		Value: values,
	}
}

// NewBreak produces a Break FlowChange.
func NewBreak() Value {
	return FlowChange{Type: Break}
}

// NewContinue produces a Continue FlowChange.
func NewContinue() Value {
	return FlowChange{Type: Continue}
}

// Expr is a thing that can be evaluated.
type Expr func(Context, ...Value) (Value, error)

// Noop is a no-operation Expr.
func Noop(c Context, args ...Value) (Value, error) {
	return nil, nil
}

// CompilerFunc is a function that converts a Node to an Expr.
type CompilerFunc func(node parser.Node) (Expr, error)

var (
	// compilerForType maps node Type to CompilerFunc.
	compilerForType [lex.TypeCount]CompilerFunc
)

func init() {
	compilerForType = [lex.TypeCount]CompilerFunc{
		lex.LeftBrace:         compileBlock,
		lex.Ident:             compileIdent,
		lex.Number:            compileNumber,
		lex.Plus:              compilePlus,
		lex.BacktickString:    compileString,
		lex.DoubleQuoteString: compileString,
		lex.SingleQuoteString: compileString,
	}
}

// Compile converts a parsed Node into an Expr.
func Compile(node parser.Node) (Expr, error) {

	c := compilerForType[node.Type()]
	if c == nil {
		return nil, fmt.Errorf("cannot compile %s", node)
	}

	return c(node)
}

func compilePlus(node parser.Node) (Expr, error) {

	left, err := Compile(node.Children[0])
	if err != nil {
		return nil, err
	}
	right, err := Compile(node.Children[1])
	if err != nil {
		return nil, err
	}

	return func(ctx Context, vals ...Value) (Value, error) {
		lVal, err := left(ctx)
		if err != nil {
			return nil, err
		}
		rVal, err := right(ctx)
		if err != nil {
			return nil, err
		}

		if v1, v2, ok := gotInts(lVal, rVal); ok {
			return v1 + v2, nil
		}
		if v1, v2, ok := gotFloats(lVal, rVal); ok {
			return v1 + v2, nil
		}
		if v1, v2, ok := gotStrings(lVal, rVal); ok {
			return v1 + v2, nil
		}

		return nil, node.Error(fmt.Errorf("cannot apply operator to argument types %T, %T", lVal, rVal))
	}, nil
}

func gotInts(i, j interface{}) (int64, int64, bool) {

	switch ii := i.(type) {
	case int64:
		switch jj := j.(type) {
		case int64:
			return ii, jj, true
		}
	}

	return 0, 0, false
}

func gotFloats(i, j Value) (float64, float64, bool) {

	var iv, jv float64

	switch ii := i.(type) {
	case int64:
		iv = float64(ii)
	case float64:
		iv = ii
	default:
		return 0.0, 0.0, false
	}

	switch jj := j.(type) {
	case int64:
		jv = float64(jj)
	case float64:
		jv = jj
	default:
		return 0.0, 0.0, false
	}

	return iv, jv, true
}

func gotStrings(i, j Value) (string, string, bool) {

	switch ii := i.(type) {
	case string:
		switch jj := j.(type) {
		case string:
			return ii, jj, true
		}
	}

	return "", "", false
}

func compileBlock(node parser.Node) (Expr, error) {

	stmts := []Expr{}

	for _, n := range node.Children {
		e, err := Compile(n)
		if err != nil {
			return nil, err
		}

		stmts = append(stmts, e)
	}

	return func(ctx Context, vals ...Value) (Value, error) {

		var lastVal interface{}
		var err error

		for _, e := range stmts {
			lastVal, err = e(ctx)
			if err != nil {
				return nil, err
			}
			if flowChange(lastVal) != None {
				return lastVal, nil
			}
		}

		return NewTuple(true, lastVal), nil
	}, nil
}

func valFunc(val Value) func(Context, ...Value) (Value, error) {
	return func(Context, ...Value) (Value, error) {
		return val, nil
	}
}

func compileIdent(node parser.Node) (Expr, error) {

	switch node.Item.Value {
	case "true":
		return valFunc(true), nil
	case "false":
		return valFunc(false), nil
	case "nil":
		return valFunc(nil), nil
	}

	return Noop, fmt.Errorf("%s:%d:%d failed to compile identifier: %s",
		node.Item.Name(), node.Item.Line, node.Item.Column, node.Item.Value)
}

func compileNumber(node parser.Node) (Expr, error) {

	i, err := strconv.ParseInt(node.Item.Value, 10, 64)
	if err == nil {
		return valFunc(i), nil
	}

	f, err := strconv.ParseFloat(node.Item.Value, 64)
	if err == nil {
		return valFunc(f), nil
	}

	return Noop, fmt.Errorf("%s:%d:%d failed to convert number: %s",
		node.Item.Name(), node.Item.Line, node.Item.Column, node.Item.Value)
}

func compileString(node parser.Node) (Expr, error) {

	s, err := strconv.Unquote(node.Item.Value)
	if err != nil {
		return nil, fmt.Errorf("%s:%d:%d failed to convert string %s: %v",
			node.Item.Name(), node.Item.Line, node.Item.Column, node.Item.Value, err)
	}

	return valFunc(s), nil
}
