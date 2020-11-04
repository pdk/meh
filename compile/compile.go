package compile

import (
	"fmt"
	"strconv"

	"github.com/pdk/meh/lex"
	"github.com/pdk/meh/parser"
)

// Convert a parse tree to an executable function

// Expr is a thing that can be evaluated.
type Expr func(*Context, ...Value) (Value, error)

// Noop is a no-operation Expr.
func Noop(ctx *Context, args ...Value) (Value, error) {
	return nil, nil
}

// CompilerFunc is a function that converts a Node to an Expr.
type CompilerFunc func(node parser.Node) (Expr, error)

var (
	// compilerForType maps node Type to CompilerFunc.
	compilerForType [lex.TypeCount]CompilerFunc
)

type binaryOps struct {
	intOp    func(int64, int64) Value
	floatOp  func(float64, float64) Value
	stringOp func(string, string) Value
}

func init() {
	compilerForType = [lex.TypeCount]CompilerFunc{
		lex.LeftBrace:         compileBlock,
		lex.Ident:             compileIdent,
		lex.Nil:               fixedValue(nil),
		lex.True:              fixedValue(true),
		lex.False:             fixedValue(false),
		lex.Continue:          fixedValue(NewContinue()),
		lex.Break:             fixedValue(NewBreak()),
		lex.Return:            compileReturn,
		lex.Function:          compileFunction,
		lex.FuncApply:         compileFuncApply,
		lex.Assign:            compileAssign,
		lex.Number:            compileNumber,
		lex.BacktickString:    compileString,
		lex.DoubleQuoteString: compileString,
		lex.SingleQuoteString: compileString,
		lex.And:               compileAnd,
		lex.Or:                compileOr,
		lex.Plus: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:    func(i, j int64) Value { return i + j },
				floatOp:  func(i, j float64) Value { return i + j },
				stringOp: func(i, j string) Value { return i + j },
			})
		},
		lex.Minus: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:   func(i, j int64) Value { return i - j },
				floatOp: func(i, j float64) Value { return i - j },
			})
		},
		lex.Mult: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:   func(i, j int64) Value { return i * j },
				floatOp: func(i, j float64) Value { return i * j },
			})
		},
		lex.Div: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:   func(i, j int64) Value { return i / j },
				floatOp: func(i, j float64) Value { return i / j },
			})
		},
		lex.Modulo: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp: func(i, j int64) Value { return i % j },
			})
		},
		lex.Equal: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:    func(i, j int64) Value { return i == j },
				floatOp:  func(i, j float64) Value { return i == j },
				stringOp: func(i, j string) Value { return i == j },
			})
		},
		lex.NotEqual: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:    func(i, j int64) Value { return i != j },
				floatOp:  func(i, j float64) Value { return i != j },
				stringOp: func(i, j string) Value { return i != j },
			})
		},
		lex.Greater: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:    func(i, j int64) Value { return i > j },
				floatOp:  func(i, j float64) Value { return i > j },
				stringOp: func(i, j string) Value { return i > j },
			})
		},
		lex.GreaterOrEqual: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:    func(i, j int64) Value { return i >= j },
				floatOp:  func(i, j float64) Value { return i >= j },
				stringOp: func(i, j string) Value { return i >= j },
			})
		},
		lex.Less: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:    func(i, j int64) Value { return i < j },
				floatOp:  func(i, j float64) Value { return i < j },
				stringOp: func(i, j string) Value { return i < j },
			})
		},
		lex.LessOrEqual: func(node parser.Node) (Expr, error) {
			return compileBinaryOp(node, binaryOps{
				intOp:    func(i, j int64) Value { return i <= j },
				floatOp:  func(i, j float64) Value { return i <= j },
				stringOp: func(i, j string) Value { return i <= j },
			})
		},
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

func compileReturn(node parser.Node) (Expr, error) {

	if len(node.Children) == 0 {
		return valFunc(NewReturn()), nil
	}

	expr, err := Compile(node.Children[0])
	if err != nil {
		return nil, err
	}

	return func(ctx *Context, vals ...Value) (Value, error) {

		val, err := expr(ctx)
		if err != nil {
			return nil, err
		}

		return NewReturn(val), nil
	}, nil
}

func compileFuncApply(node parser.Node) (Expr, error) {

	fn, err := Compile(node.Children[0])
	if err != nil {
		return nil, err
	}

	args := []Expr{}
	for _, e := range node.Children[1].Children {
		next, err := Compile(e)
		if err != nil {
			return nil, err
		}

		args = append(args, next)
	}

	return func(ctx *Context, vals ...Value) (Value, error) {

		fnVal, err := fn(ctx)
		if err != nil {
			return nil, err
		}

		expr, ok := fnVal.(func(*Context, ...Value) (Value, error))
		if !ok {
			return nil, fmt.Errorf("cannot invoke non-function: %T %v", fnVal, fnVal)
		}

		argValues := []Value{}
		for _, a := range args {
			nextVal, err := a(ctx)
			if err != nil {
				return nil, err
			}

			argValues = append(argValues, nextVal)
		}

		return expr(ctx, argValues...)
	}, nil
}

func compileFunction(node parser.Node) (Expr, error) {

	if len(node.Children) != 2 {
		return nil, node.Error(fmt.Errorf("malformed function: requires param list & body"))
	}

	params, err := parameterNames(node.Children[0])
	if err != nil {
		return nil, err
	}

	body := node.Children[1]
	if !body.Type().Match(lex.LeftBrace) {
		return nil, node.Error(fmt.Errorf("malformed function: requires block"))
	}

	block, err := Compile(body)
	if err != nil {
		return nil, err
	}

	return func(ctx *Context, vals ...Value) (Value, error) {
		return func(ctx *Context, vals ...Value) (Value, error) {

			if len(vals) != len(params) {
				return nil, fmt.Errorf("failed to apply function: received %d arguments for %d parameters", len(vals), len(params))
			}

			funcCtx := NewContext(ctx)
			for i, p := range params {
				_, err := funcCtx.Set(p, vals[i])
				if err != nil {
					return nil, err
				}
			}

			return block(funcCtx)
		}, nil
	}, nil
}

func parameterNames(node parser.Node) ([]string, error) {

	if !node.Type().Match(lex.LeftParen) {
		return nil, node.Error(fmt.Errorf("malformed function, parameter list required"))
	}

	names := []string{}
	for _, next := range node.Children {
		if !next.Type().Match(lex.Ident) {
			return nil, node.Error(fmt.Errorf("malformed function, parameters must be identifiers, found %v", next))
		}

		names = append(names, next.Item.Value)
	}

	return names, nil
}

func compileAssign(node parser.Node) (Expr, error) {

	if len(node.Children) != 2 {
		return nil, node.Error(fmt.Errorf("assignment requires exactly 2 children"))
	}

	lhs := node.Children[0]
	if !lhs.Type().Match(lex.Ident) {
		return nil, node.Error(fmt.Errorf("assignment requires an identifier"))
	}
	left := lhs.Item.Value

	right, err := Compile(node.Children[1])
	if err != nil {
		return nil, err
	}

	return func(ctx *Context, vals ...Value) (Value, error) {

		val, err := right(ctx)
		if err != nil {
			return nil, err
		}

		return ctx.Set(left, val)

	}, nil
}

func compileAnd(node parser.Node) (Expr, error) {

	left, err := Compile(node.Children[0])
	if err != nil {
		return nil, err
	}
	right, err := Compile(node.Children[1])
	if err != nil {
		return nil, err
	}

	return func(ctx *Context, vals ...Value) (Value, error) {
		lVal, err := left(ctx)
		if err != nil {
			return nil, err
		}

		if !isTruthy(lVal) {
			return lVal, nil
		}

		rVal, err := right(ctx)
		if err != nil {
			return nil, err
		}

		return rVal, nil
	}, nil
}

func compileOr(node parser.Node) (Expr, error) {
	left, err := Compile(node.Children[0])
	if err != nil {
		return nil, err
	}
	right, err := Compile(node.Children[1])
	if err != nil {
		return nil, err
	}

	return func(ctx *Context, vals ...Value) (Value, error) {
		lVal, err := left(ctx)
		if err != nil {
			return nil, err
		}

		if isTruthy(lVal) {
			return lVal, nil
		}

		rVal, err := right(ctx)
		if err != nil {
			return nil, err
		}

		return rVal, nil
	}, nil
}

// isTruthy returns the boolean value of a boolean input. For a tuple, return
// isTruthy of the first element in the tuple. Everything else is true.
func isTruthy(v Value) bool {

	if b, ok := v.(bool); ok {
		return b
	}

	if t, ok := v.(Tuple); ok {
		return isTruthy(t.Values[0])
	}

	return true
}

func compileBinaryOp(node parser.Node, ops binaryOps) (Expr, error) {

	left, err := Compile(node.Children[0])
	if err != nil {
		return nil, err
	}
	right, err := Compile(node.Children[1])
	if err != nil {
		return nil, err
	}

	return func(ctx *Context, vals ...Value) (Value, error) {
		lVal, err := left(ctx)
		if err != nil {
			return nil, err
		}
		rVal, err := right(ctx)
		if err != nil {
			return nil, err
		}

		if ops.intOp != nil {
			if v1, v2, ok := gotInts(lVal, rVal); ok {
				return ops.intOp(v1, v2), nil
			}
		}

		if ops.floatOp != nil {
			if v1, v2, ok := gotFloats(lVal, rVal); ok {
				return ops.floatOp(v1, v2), nil
			}
		}

		if ops.stringOp != nil {
			if v1, v2, ok := gotStrings(lVal, rVal); ok {
				return ops.stringOp(v1, v2), nil
			}
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

	return func(ctx *Context, vals ...Value) (Value, error) {

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

func valFunc(val Value) func(*Context, ...Value) (Value, error) {
	return func(*Context, ...Value) (Value, error) {
		return val, nil
	}
}

func fixedValue(val Value) func(node parser.Node) (Expr, error) {
	return func(node parser.Node) (Expr, error) {
		return valFunc(val), nil
	}
}

func compileIdent(node parser.Node) (Expr, error) {
	return func(ctx *Context, args ...Value) (Value, error) {
		return ctx.Get(node.Item.Value), nil
	}, nil
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
