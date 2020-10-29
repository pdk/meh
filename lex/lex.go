package lex

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	tabWidth = 4
)

// Lexer produces lexemes aka items.
type Lexer struct {
	name         string
	input        io.Reader
	scanner      *bufio.Scanner
	backupBuffer chan fetch
	current      strings.Builder
	line         int
	col          int
	items        chan Item
	lastItem     Item
}

type fetch struct {
	val rune
	err error
}

// Name returns the name of the input of the lexer.
func (l *Lexer) Name() string {
	if l == nil {
		return "unknown"
	}
	return l.name
}

// New creates a new lexer.
func New(name string, input io.Reader) (*Lexer, chan Item) {
	s := bufio.NewScanner(input)
	s.Split(bufio.ScanRunes)

	l := &Lexer{
		name:         name,
		input:        input,
		scanner:      s,
		backupBuffer: make(chan fetch, 2),
		items:        make(chan Item),
		line:         1,
		col:          1,
	}

	go l.run()

	return l, l.items
}

const eof = -1

// next returns the next rune. returns empty string ("") when no more input.
func (l *Lexer) next() (rune, error) {
	select {
	case next := <-l.backupBuffer:
		return next.val, next.err
	default:
		advanced := l.scanner.Scan()
		if !advanced {
			return eof, l.scanner.Err()
		}
		r, _ := utf8.DecodeRune(l.scanner.Bytes())
		return r, l.scanner.Err()
	}
}

// backup returns a rune to the input queue.
func (l *Lexer) backup(r rune, err error) {
	l.backupBuffer <- fetch{r, err}
}

// peek returns the upcoming rune/string without advancing.
func (l *Lexer) peek() rune {
	n, err := l.next()
	l.backup(n, err)
	return n
}

// run consumes input to produce items.
func (l *Lexer) run() {
	defer func() {
		// log.Printf("lexer closing")
		close(l.items)
	}()

	for state := cleanSlate; state != nil; {
		state = state(l)
	}

	l.emit(EOF)

	// log.Printf("lexer run complete")
}

func (l *Lexer) advancePos(s string) {
	// log.Printf("advancing: %q", s)
	var last rune
	for _, r := range s {
		if r == '\t' {
			l.col++
			l.col = l.col + (l.col % tabWidth)
		}

		if r == '\n' || (r == '\r' && last != '\n') {
			l.line++
			l.col = 0
		}
		l.col++

		last = r
	}
}

// emit sends an Item down the channel.
func (l *Lexer) emit(t Type) {
	line, col, s := l.line, l.col, l.current.String()
	l.advancePos(s)
	l.current.Reset()

	i := Item{
		Lexer:  l,
		Type:   t,
		Value:  s,
		Line:   line,
		Column: col,
	}

	if i.Type != HashComment && i.Type != SlashComment {
		l.lastItem = i
	}

	l.items <- i
}

func (l *Lexer) emitError(mesg string) {
	line, col, s := l.line, l.col, l.current.String()
	l.advancePos(s)
	l.current.Reset()

	l.items <- Item{
		Lexer:  l,
		Type:   Error,
		Value:  s,
		Error:  mesg,
		Line:   line,
		Column: col,
	}
}

func (l *Lexer) collect(r rune) {
	l.current.WriteRune(r)
}

type stateFunc func(*Lexer) stateFunc

// cleanSlate is scanning we-don't-know-what-yet
func cleanSlate(l *Lexer) stateFunc {

	r, err := l.next()
	if err != nil {
		l.emitError(fmt.Sprintf("failed to scan next rune at %d:%d: %v", l.line, l.col, err))
		return nil
	}

	l.collect(r)

	// First, handle the cases that do not require peeking

	switch r {
	case eof:
		return nil
	case '\t', '\n', '\v', '\f', '\r', ' ':
		l.maybeEmitSeparator(r)
		return whitespace
	case '#':
		return hashComment
	case '"':
		return doubleQuoteString
	case '\'':
		return singleQuoteString
	case '`':
		return backtickString
	case
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return number
	}

	// Second, handle the cases where peeking is required.

	p := l.peek()

	switch r {
	case '-':
		if '0' <= p && p <= '9' {
			return number
		}
	case '/':
		if p == '/' {
			return slashComment
		}
	}

	op := doubleRuneOperator(r, p)
	if op != Error {
		r, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan double rune operator at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		l.collect(r)
		l.emit(op)

		return cleanSlate
	}

	op = singleRuneOperator(r)
	if op != Error {
		l.emit(op)

		return cleanSlate
	}

	if isLetter(r) {
		return word
	}

	l.emitError("unrecognized rune")
	return nil
}

func word(l *Lexer) stateFunc {
	for {
		r, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan word at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		if isLetter(r) || isDigit(r) {
			l.collect(r)
			continue
		}

		l.backup(r, nil)

		l.emit(Ident)

		return cleanSlate
	}
}

func number(l *Lexer) stateFunc {
	for {
		r, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan number at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		if '0' <= r && r <= '9' {
			l.collect(r)
			continue
		}

		l.backup(r, nil)

		l.emit(Number)

		return cleanSlate
	}
}

func (l *Lexer) maybeEmitSeparator(r rune) {
	switch r {
	case '\n', '\r', '\v', '\f':
		switch l.lastItem.Type {
		case Ident, Number, DoubleQuoteString,
			SingleQuoteString, BacktickString,
			RightParen,
			RightBrace: // unclear if RightBrace should be here

			l.emit(Separator)
		}
	}
}

func whitespace(l *Lexer) stateFunc {
	for {
		n, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan whitespace at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		switch n {
		case '\n', '\r', '\v', '\f', '\t', ' ':
			l.collect(n)
			l.maybeEmitSeparator(n)
			continue
		}

		// count how much space we ate
		l.advancePos(l.current.String())
		l.current.Reset()

		l.backup(n, nil)
		return cleanSlate
	}
}

// hashComment reads until the end of the line.
func hashComment(l *Lexer) stateFunc {
	for {
		n, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan within comment at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		l.collect(n)

		if n == '\n' || n == '\r' || n == eof {
			l.emit(HashComment)
			return cleanSlate
		}
	}
}

// slashComment reads until the end of the line.
func slashComment(l *Lexer) stateFunc {
	for {
		n, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan within comment at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		l.collect(n)

		if n == '\n' || n == '\r' || n == eof {
			l.emit(SlashComment)
			return cleanSlate
		}
	}
}

// doubleQuoteString scans a doublequote delimited string.
func doubleQuoteString(l *Lexer) stateFunc {
	for {
		n, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan within string at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		if n == '\n' || n == '\r' || n == eof {
			l.emitError(fmt.Sprintf("unclosed double quote string at %d:%d", l.line, l.col))
			return nil
		}

		if n == '\\' {
			l.collect(n)

			n, err := l.next()
			if err != nil {
				l.emitError(fmt.Sprintf("failed to scan within string at %d:%d: %v", l.line, l.col, err))
				return nil
			}

			l.collect(n)
			continue
		}

		l.collect(n)

		if n == '"' {
			l.emit(DoubleQuoteString)
			return cleanSlate
		}
	}
}

// singleQuoteString scans a single quote delimited string.
func singleQuoteString(l *Lexer) stateFunc {
	for {
		n, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan within string at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		if n == eof {
			l.emitError(fmt.Sprintf("unclosed single quote string at %d:%d", l.line, l.col))
		}

		if n == '\\' {
			l.collect(n)

			n, err := l.next()
			if err != nil {
				l.emitError(fmt.Sprintf("failed to scan within string at %d:%d: %v", l.line, l.col, err))
				return nil
			}

			l.collect(n)
			continue
		}

		l.collect(n)

		if n == '\'' {
			l.emit(SingleQuoteString)
			return cleanSlate
		}
	}
}

// backtickString scans a back tick delimited string.
func backtickString(l *Lexer) stateFunc {
	for {
		n, err := l.next()
		if err != nil {
			l.emitError(fmt.Sprintf("failed to scan within string at %d:%d: %v", l.line, l.col, err))
			return nil
		}

		if n == eof {
			l.emitError(fmt.Sprintf("unclosed backtick string at %d:%d", l.line, l.col))
			return nil
		}

		l.collect(n)

		if n == '`' {
			l.emit(BacktickString)
			return cleanSlate
		}
	}
}

// below copied from https://golang.org/src/go/scanner/scanner.go

func isLetter(ch rune) bool {
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return isDecimal(ch) || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func lower(ch rune) rune {
	// returns lower-case ch iff ch is ASCII letter
	return ('a' - 'A') | ch
}

func isDecimal(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isHex(ch rune) bool {
	return '0' <= ch && ch <= '9' || 'a' <= lower(ch) && lower(ch) <= 'f'
}
