package main

import (
	"log"
	"os"

	"github.com/pdk/meh/lex"
)

func main() {

	l, c := lex.New("stdin", os.Stdin)

	log.Printf("new lexer: %v", l)

	for item := range c {
		log.Printf("%s:%3d:%3d %-20s %q",
			item.Lexer.Name(),
			item.Line,
			item.Column,
			item.Type,
			item.Value)
	}
}
