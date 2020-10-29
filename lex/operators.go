package lex

func singleRuneOperator(r rune) Type {
	switch r {
	case ';':
		return Separator
	case ',':
		return Comma
	case '+':
		return Plus
	case '-':
		return Minus
	case '*':
		return Mult
	case '/':
		return Div
	case '%':
		return Modulo
	case '<':
		return Less
	case '>':
		return Greater
	case '!':
		return Not
	case '.':
		return Dot
	case '=':
		return Assign
	case '(':
		return LeftParen
	case ')':
		return RightParen
	case '{':
		return LeftBrace
	case '}':
		return RightBrace
	}

	return Error
}
func doubleRuneOperator(r1, r2 rune) Type {

	if r1 == '>' {
		switch r2 {
		case '>':
			return Pipe
		case '=':
			return GreaterOrEqual
		}
	}

	if r2 == '=' {
		switch r1 {
		case '!':
			return NotEqual
		case '=':
			return Equal
		case ':':
			return Assign
		case '+':
			return PlusAssign
		case '-':
			return MinusAssign
		case '*':
			return MultAssign
		case '/':
			return DivAssign
		case '%':
			return ModuloAssign
		case '<':
			return LessOrEqual
		}
	}

	if r1 == '|' && r2 == '|' {
		return Or
	}

	if r1 == '&' && r2 == '&' {
		return And
	}

	return Error
}
