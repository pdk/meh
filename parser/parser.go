package parser

import (
	"io"
	"log"
	"strings"

	"github.com/pdk/meh/lex"
)

// Parser handles parsing a stream of input
type Parser struct {
	lexer *lex.Lexer
	items chan lex.Item
	// itemBuf []lex.Item
}

// NewFromReader creates a parser for an input stream.
func NewFromReader(name string, reader io.Reader) *Parser {

	lexer, items := lex.New(name, reader)

	return &Parser{
		lexer: lexer,
		items: items,
	}
}

// NewFromString creates a parser for a single string.
func NewFromString(name, input string) *Parser {
	return NewFromReader(name, strings.NewReader(input))
}

// Node is a node in the parse tree.
type Node struct {
	Item     lex.Item
	Resolved bool `json:"-"` // marker for "parsed"
	Children []Node
}

// Type returns the lex.Type of the Node.
func (n Node) Type() lex.Type {
	return n.Item.Type
}

func (n Node) Error(err error) error {
	return n.Item.Error(err)
}

func (n Node) String() string {
	s := strings.Builder{}

	if len(n.Children) > 0 {
		s.WriteString("(")
	}

	s.WriteString(n.Item.Type.String())
	s.WriteString("<")
	s.WriteString(n.Item.Value)
	s.WriteString(">")

	for _, c := range n.Children {
		s.WriteString(" ")
		s.WriteString(c.String())
	}

	if len(n.Children) > 0 {
		s.WriteString(")")
	}

	return s.String()
}

// Parse will parse the complete input, and return an AST.
func (p *Parser) Parse() Node {
	prog := lex.Item{
		Lexer:  p.lexer,
		Type:   lex.LeftBrace,
		Value:  "{",
		Line:   1,
		Column: 1,
	}

	return parseItems(prog, nodify(noComment(p.items)))
}

func parseItems(wrapItem lex.Item, items chan Node) Node {

	stmts := []Node{}

	for x := range pipeline(
		slicify(bracify(parenthify(items))),
		binaryOps(lex.Mult, lex.Div, lex.Modulo),
		binaryOps(lex.Plus, lex.Minus),
		binaryOps(lex.Less, lex.Greater, lex.LessOrEqual, lex.GreaterOrEqual, lex.Equal, lex.NotEqual),
		binaryOps(lex.And, lex.Or),
		binaryOps(lex.Comma),
		collapse(lex.Comma),
		binaryOpsRightToLeft(lex.Assign, lex.PlusAssign, lex.MinusAssign, lex.MultAssign, lex.DivAssign, lex.ModuloAssign),
		reassign,
		checkResolved,
	) {

		if len(x) > 1 {
			n := x[0]
			log.Printf("error parsing statement near %s:%d:%d : %v", n.Item.Name(), n.Item.Line, n.Item.Column, x)
			continue
		}
		if len(x) == 0 {
			log.Printf("parser received statment with 0 elements (very bad!)")
			continue
		}

		stmts = append(stmts, x[0])
	}

	return Node{
		Item:     wrapItem,
		Resolved: true,
		Children: stmts,
	}
}

func checkResolved(stmt []Node) []Node {

	for _, node := range stmt {
		if !node.Resolved {
			log.Printf("%s:%d:%d misplaced operator/missing operand %q",
				node.Item.Name(), node.Item.Line, node.Item.Column,
				node.Item.Value)
		}
		checkResolved(node.Children)
	}

	return stmt
}

func reassign(stmt []Node) []Node {

	// [+= x y] => [= x [+ x y]]
	for i, n := range stmt {

		newOp := assignOp(n.Type())
		if newOp.Match(lex.Error) {
			continue
		}

		opNode := Node{
			Item:     n.Item,
			Resolved: n.Resolved,
			Children: []Node{
				n.Children[0],
				n.Children[1],
			},
		}
		opNode.Item.Type = newOp

		newNode := Node{
			Item:     n.Item,
			Resolved: n.Resolved,
			Children: []Node{
				n.Children[0],
				opNode,
			},
		}
		newNode.Item.Type = lex.Assign

		return append(append(stmt[:i], newNode), stmt[i+1:]...)
	}

	return stmt
}

func assignOp(op lex.Type) lex.Type {
	switch op {
	case lex.PlusAssign:
		return lex.Plus
	case lex.MinusAssign:
		return lex.Minus
	case lex.MultAssign:
		return lex.Mult
	case lex.DivAssign:
		return lex.Div
	case lex.ModuloAssign:
		return lex.Modulo
	}

	return lex.Error
}

func binaryOps(operators ...lex.Type) func(stmt []Node) []Node {

	var f func(stmt []Node) []Node
	f = func(stmt []Node) []Node {

		// [... x * y ...] => [... {* [x y]} ...]
		for i := 0; i < len(stmt)-2; i++ {
			if unresolvedType(stmt[i+1]).Match(operators...) {
				operation := Node{
					Resolved: true,
					Item:     stmt[i+1].Item,
					Children: []Node{stmt[i], stmt[i+2]},
				}
				return f(gorp(stmt[:i], operation, stmt[i+3:]))
			}
		}

		return stmt
	}

	return f
}

func binaryOpsRightToLeft(operators ...lex.Type) func(stmt []Node) []Node {

	var f func(stmt []Node) []Node
	f = func(stmt []Node) []Node {

		// [... x * y ...] => [... {* [x y]} ...]
		for i := len(stmt) - 2; i >= 0; i-- {
			if unresolvedType(stmt[i+1]).Match(operators...) {
				operation := Node{
					Resolved: true,
					Item:     stmt[i+1].Item,
					Children: []Node{stmt[i], stmt[i+2]},
				}
				return f(gorp(stmt[:i], operation, stmt[i+3:]))
			}
		}

		return stmt
	}

	return f
}

func collapse(operators ...lex.Type) func(stmt []Node) []Node {

	var f func(stmt []Node) []Node
	f = func(stmt []Node) []Node {

		// [ * [ * ... ] ... ] => [ * ... ... ]
		for i := 0; i < len(stmt); i++ {

			op := stmt[i]
			if len(op.Children) == 0 || !op.Type().Match(operators...) {
				continue
			}

			childOp := op.Children[0]
			if op.Type() != childOp.Type() {
				continue
			}

			op.Children = append(childOp.Children, op.Children[1:]...)

			return f(gorp(stmt[:i], op, stmt[i+1:]))
		}

		return stmt
	}

	return f
}

func unresolvedType(n Node) lex.Type {
	if n.Resolved {
		return lex.Nada
	}
	return n.Item.Type
}

func gorp(before []Node, middle Node, after []Node) []Node {
	return append(append(before, middle), after...)
}

func pipeline(in chan []Node, jobs ...func([]Node) []Node) chan []Node {

	if len(jobs) == 0 {
		return in
	}

	f := jobs[0]
	out := make(chan []Node)

	go func() {
		defer close(out)

		for stmt := range in {
			out <- f(stmt)
		}
	}()

	return pipeline(out, jobs[1:]...)
}

func slicify(in chan Node) chan []Node {
	out := make(chan []Node)

	go func() {
		defer close(out)

		slice := make([]Node, 0)
		for n := range in {
			if n.Item.Type == lex.Separator || n.Item.Type == lex.EOF {
				if len(slice) > 0 {
					out <- slice
					slice = make([]Node, 0)
				}
				continue
			}

			slice = append(slice, n)
		}

		if len(slice) > 0 {
			out <- slice
		}
	}()

	return out
}

func bracify(in chan Node) chan Node {

	out := make(chan Node)

	go func() {
		defer close(out)

		for n := range in {

			if !n.Item.Type.Match(lex.LeftBrace) {
				out <- n
				continue
			}

			sub := make(chan Node)

			go func(openBrace Node) {
				defer close(sub)

				depth := 1
				for n := range in {
					depth = depth + adjustDepth(n, lex.LeftBrace, lex.RightBrace)

					switch {
					case depth == 0:
						return
					case n.Item.Type.Match(lex.EOF):
						log.Printf("%s:%d:%d open brace without close %q",
							openBrace.Item.Name(), openBrace.Item.Line, openBrace.Item.Column,
							openBrace.Item.Value)
						return
					}

					sub <- n
				}
			}(n)

			out <- parseItems(n.Item, sub)
		}
	}()

	return out
}

func parenthify(in chan Node) chan Node {

	out := make(chan Node)

	go func() {
		defer close(out)

		for n := range in {

			if !n.Item.Type.Match(lex.LeftParen) {
				out <- n
				continue
			}

			sub := make(chan Node)

			go func(openParen Node) {
				defer close(sub)

				depth := 1
				for n := range in {
					depth = depth + adjustDepth(n, lex.LeftParen, lex.RightParen)

					switch {
					case depth == 0:
						return
					case n.Item.Type.Match(lex.EOF):
						log.Printf("%s:%d:%d open paren without close %q",
							openParen.Item.Name(), openParen.Item.Line, openParen.Item.Column,
							openParen.Item.Value)
						return
					}

					sub <- n
				}
			}(n)

			out <- parseItems(n.Item, sub)
		}
	}()

	return out
}

func adjustDepth(n Node, open, close lex.Type) int {
	if close.Match(n.Item.Type) {
		return -1
	}
	if open.Match(n.Item.Type) {
		return 1
	}
	return 0
}

func nodify(in chan lex.Item) chan Node {
	out := make(chan Node)

	go func() {
		defer close(out)

		for item := range in {
			out <- Node{
				Item: item,
				Resolved: item.Type.Match(
					lex.Ident, lex.Number,
					lex.DoubleQuoteString, lex.SingleQuoteString, lex.BacktickString),
			}
		}
	}()

	return out
}

func noComment(in chan lex.Item) chan lex.Item {
	out := make(chan lex.Item)

	go func() {
		defer close(out)

		for next := range in {
			if next.Type == lex.HashComment || next.Type == lex.SlashComment {
				continue
			}

			out <- next
		}
	}()

	return out
}
