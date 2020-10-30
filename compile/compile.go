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
type Expr func(Context, ...Value) Value

// Noop is a no-operation Expr.
func Noop(c Context, args ...Value) Value {
	return nil
}

// Compile converts a parsed Node into an Expr.
func Compile(node parser.Node) (Expr, error) {

	switch node.Type() {
	case lex.LeftBrace:
		return compileBlock(node)
	case lex.Ident:
		return compileIdent(node)
	case lex.Number:
		return compileNumber(node)
	case lex.Plus:
		return compilePlus(node)
	case lex.BacktickString, lex.DoubleQuoteString, lex.SingleQuoteString:
		return compileString(node)
	}

	return Noop, nil
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

	return func(ctx Context, vals ...Value) Value {
		lVal := left(ctx)
		rVal := right(ctx)

		if v1, v2, ok := gotInts(lVal, rVal); ok {
			return v1 + v2
		}
		if v1, v2, ok := gotFloats(lVal, rVal); ok {
			return v1 + v2
		}
		if v1, v2, ok := gotStrings(lVal, rVal); ok {
			return v1 + v2
		}

		return nil
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

	return func(ctx Context, vals ...Value) Value {

		var lastVal interface{}
		for _, e := range stmts {
			lastVal = e(ctx)
			if flowChange(lastVal) != None {
				return lastVal
			}
		}

		return NewTuple(true, lastVal)
	}, nil
}

func compileIdent(node parser.Node) (Expr, error) {

	switch node.Item.Value {
	case "true":
		return func(Context, ...Value) Value {
			return true
		}, nil
	case "false":
		return func(Context, ...Value) Value {
			return false
		}, nil
	case "nil":
		return func(Context, ...Value) Value {
			return nil
		}, nil
	}

	return Noop, fmt.Errorf("%s:%d:%d failed to compile identifier: %s",
		node.Item.Name(), node.Item.Line, node.Item.Column, node.Item.Value)
}

func compileNumber(node parser.Node) (Expr, error) {

	i, err := strconv.ParseInt(node.Item.Value, 10, 64)
	if err == nil {
		return func(Context, ...Value) Value {
			return i
		}, nil
	}

	f, err := strconv.ParseFloat(node.Item.Value, 64)
	if err == nil {
		return func(Context, ...Value) Value {
			return f
		}, nil
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

	return func(Context, ...Value) Value {
		return s
	}, nil
}
