package lex

import "encoding/json"

// Item is produced by a lexer.
type Item struct {
	*Lexer
	Type
	Value  string
	Error  string
	Line   int
	Column int
}

// func (i Item) String() string {
// 	return fmt.Sprintf("[%s %s]", i.Type, i.Value)
// }

// Type is the kind of the Item.
type Type uint

// Match checks if a type matches any of a set of types.
func (t Type) Match(targets ...Type) bool {
	for _, o := range targets {
		if t == o {
			return true
		}
	}
	return false
}

// func (t Type) MarshalJSON() ([]byte, error) {
// 	return []byte(fmt.Sprintf("%q", t.String())), nil
// }

// MarshalJSON helps Item -> JSON
func (i Item) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"Type":  i.Type.String(),
		"Value": i.Value,
	}

	return json.Marshal(m)
}

// Define the known Item Types.
const (
	EOF Type = iota
	Nada
	Error
	// expr separator
	Separator
	// identifiers
	Ident
	// literal values
	Number
	DoubleQuoteString
	SingleQuoteString
	BacktickString
	// comments
	HashComment
	SlashComment
	// code block
	LeftBrace
	RightBrace
	// parens
	LeftParen
	RightParen
	// prefix operators
	Not
	// infix operators
	Comma
	Plus
	Minus
	Mult
	Div
	Modulo
	Less
	Greater
	Dot
	Assign
	Pipe
	GreaterOrEqual
	NotEqual
	Equal
	PlusAssign
	MinusAssign
	MultAssign
	DivAssign
	ModuloAssign
	LessOrEqual
	Or
	And
)

// String returns string name of a Type.
func (t Type) String() string {
	switch t {
	case EOF:
		return "EOF"
	case Nada:
		return "Nada"
	case Error:
		return "Error"
	case Ident:
		return "Ident"
	case Separator:
		return "Separator"
	case Number:
		return "Number"
	case DoubleQuoteString:
		return "DoubleQuoteString"
	case SingleQuoteString:
		return "SingleQuoteString"
	case BacktickString:
		return "BacktickString"
	case HashComment:
		return "HashComment"
	case SlashComment:
		return "SlashComment"
	case LeftBrace:
		return "LeftBrace"
	case RightBrace:
		return "RightBrace"
	case Comma:
		return "Comma"
	case Plus:
		return "Plus"
	case Minus:
		return "Minus"
	case Mult:
		return "Mult"
	case Div:
		return "Div"
	case Modulo:
		return "Modulo"
	case Less:
		return "Less"
	case Greater:
		return "Greater"
	case Not:
		return "Not"
	case Dot:
		return "Dot"
	case Assign:
		return "Assign"
	case LeftParen:
		return "LeftParen"
	case RightParen:
		return "RightParen"
	case Pipe:
		return "Pipe"
	case GreaterOrEqual:
		return "GreaterOrEqual"
	case NotEqual:
		return "NotEqual"
	case Equal:
		return "Equal"
	case PlusAssign:
		return "PlusAssign"
	case MinusAssign:
		return "MinusAssign"
	case MultAssign:
		return "MultAssign"
	case DivAssign:
		return "DivAssign"
	case ModuloAssign:
		return "ModuloAssign"
	case LessOrEqual:
		return "LessOrEqual"
	case Or:
		return "Or"
	case And:
		return "And"
	}

	return "unknown"
}