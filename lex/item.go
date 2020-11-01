package lex

import (
	"encoding/json"
	"fmt"
)

// Item is produced by a lexer.
type Item struct {
	*Lexer
	Type
	Value  string
	Line   int
	Column int
	error  // perhaps there was a problem
}

// ItemError composes an Item with an error.
type ItemError struct {
	item *Item
	err  error
}

// Unwrap allows unwrapping an ItemError.
func (ierr ItemError) Unwrap() error {
	return ierr.err
}

// Error provides the standand error interface for an ItemError.
func (ierr ItemError) Error() string {
	name := ""
	if ierr.item.name != "stdin" {
		name = ierr.item.name + ":"
	}

	value := ierr.item.Value
	if len(value) > 8 {
		value = value[:5] + "..."
	}
	return fmt.Sprintf("%s%d:%d (%q) %s",
		name, ierr.item.Line, ierr.item.Column, value, ierr.err.Error())
}

func (i *Item) Error(err error) ItemError {
	return ItemError{
		item: i,
		err:  err,
	}
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
	Equal
	NotEqual
	Greater
	GreaterOrEqual
	Less
	LessOrEqual
	Dot
	Assign
	Pipe
	PlusAssign
	MinusAssign
	MultAssign
	DivAssign
	ModuloAssign
	Or
	And
	// max number of Item Types
	TypeCount
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
