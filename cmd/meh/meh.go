package main

import (
	"fmt"
	"io"
	"os"

	"github.com/pdk/meh/compile"
	"github.com/pdk/meh/parser"
)

func main() {
	if err := run(os.Args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {

	p := parser.NewFromReader("stdin", os.Stdin)

	parsed := p.Parse()
	// log.Printf("parsed: %s", parsed)

	program, err := compile.Compile(parsed)
	if err != nil {
		return err
	}

	// log.Printf("program: %#v", program)

	c := compile.Context{}

	result, err := program(c)
	if err != nil {
		fmt.Printf("%v", err)
	} else {
		fmt.Println(result)
	}
	return nil
}
