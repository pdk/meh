package main

import (
	"log"
	"os"

	"github.com/pdk/meh/parser"
)

func main() {

	p := parser.NewFromReader("stdin", os.Stdin)

	log.Printf("new parser: %v", p)

	program := p.Parse()

	// t, _ := json.MarshalIndent(program, "", "    ")
	// log.Printf("%s", t)

	log.Printf("program: %v", program)
}
