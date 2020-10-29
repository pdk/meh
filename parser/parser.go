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

func (n Node) String() string {
	s := strings.Builder{}

	if len(n.Children) > 0 {
		s.WriteString("(")
	}

	// s.WriteString(n.Item.Type.String())
	// s.WriteString("<")
	s.WriteString(n.Item.Value)
	// s.WriteString(">")

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
	return parseItems(nodify(dropComments(p.items)))
}

func parseItems(items chan Node) Node {

	stmts := []Node{}

	for x := range pipeline(
		slicify(bracify(items)),
		binaryOps(lex.Mult, lex.Div, lex.Modulo),
		binaryOps(lex.Plus, lex.Minus),
		binaryOps(lex.Less, lex.Greater, lex.LessOrEqual, lex.GreaterOrEqual, lex.Equal, lex.NotEqual),
		binaryOps(lex.And, lex.Or),
		binaryOps(lex.Assign, lex.PlusAssign, lex.MinusAssign, lex.MultAssign, lex.DivAssign, lex.ModuloAssign),
		checkResolved,
	) {
		stmts = append(stmts, x[0])
	}

	return Node{
		Item: lex.Item{
			Type:  lex.LeftBrace,
			Value: "{",
		},
		Resolved: true,
		Children: stmts,
	}
}

func checkResolved(stmt []Node) []Node {

	if len(stmt) != 1 {
		log.Printf("error parsing statement (%d): %v", len(stmt), stmt)
	}

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

func parenify(stmt []Node) []Node {
	left, right := 0, 0
	for i := 0; i < len(stmt); i++ {
		switch unresolvedType(stmt[i]) {
		case lex.LeftParen:
			left++
		case lex.RightParen:
			right++
		}
	}

	if left == 0 && right == 0 {
		return stmt
	}

	if left != right {
		// return unresolved
		return stmt
	}

	return stmt
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
		defer func() {
			close(out)
		}()

		for stmt := range in {
			out <- f(stmt)
		}
	}()

	return pipeline(out, jobs[1:]...)
}

func slicify(in chan Node) chan []Node {
	out := make(chan []Node)

	go func() {
		defer func() {
			// log.Printf("closing slicify")
			close(out)
		}()

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

			depth := 1
			p := make(chan Node)

			go func() {
				defer close(p)

				for n := range in {
					switch n.Item.Type {
					case lex.LeftBrace:
						depth++
					case lex.RightBrace:
						depth--
					}

					if depth == 0 || n.Item.Type.Match(lex.EOF) {
						p <- Node{Item: lex.Item{Type: lex.EOF}}
						break
					}

					p <- n
				}
			}()

			out <- parseItems(p)
		}
	}()

	return out
}

func nodify(in chan lex.Item) chan Node {
	out := make(chan Node)

	go func() {
		defer func() {
			// log.Printf("closing nodify")
			close(out)
		}()

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

func dropComments(in chan lex.Item) chan lex.Item {
	out := make(chan lex.Item)

	go func() {
		defer func() {
			// log.Printf("closing dropComments")
			close(out)
		}()

		for next := range in {
			if next.Type == lex.HashComment || next.Type == lex.SlashComment {
				// log.Printf("dropping comment: %s", next.Value)
				continue
			}

			// log.Printf("passing %s %s", next.Type, next.Value)

			out <- next
		}
	}()

	return out
}
