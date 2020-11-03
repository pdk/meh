package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/pdk/meh/compile"
	"github.com/pdk/meh/lex"
	"github.com/pdk/meh/parser"
)

func main() {
	if err := run(os.Args); err != nil {
		log.Fatalf("program terminated: %v", err)
	}
}

func run(args []string) error {

	if len(args) > 1 {
		fileName := args[1]

		input, err := os.Open(fileName)
		if err != nil {
			return fmt.Errorf("cannot run %s: %v", fileName, err)
		}

		return runFile(fileName, input)
	}

	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		return runREPL()
	}

	// log.Printf("running stdin")
	return runFile("stdin", os.Stdin)
}

func runREPL() error {

	fmt.Printf("meh 0.0.x\n")

	ctx := compile.NewTopContext()

	scanner := bufio.NewScanner(os.Stdin)

	var input string
	for {
		if len(input) == 0 {
			fmt.Printf("meh? ")
		} else {
			fmt.Printf("...? ")
		}

		scanned := scanner.Scan()
		if !scanned {
			return nil
		}

		nextLine := scanner.Text()
		if nextLine != "." {
			input += nextLine + "\n"
		}

		if nextLine == "." || (balanced(input) && isComplete(input)) {
			err := runProgram(ctx, "repl", strings.NewReader(input), true)
			if err != nil {
				log.Printf("%v", err)
			}
			input = ""
		}
	}
}

func isComplete(input string) bool {
	_, items := lex.New("repl", strings.NewReader(input))
	lastItem := <-items
	for item := range items {
		lastItem = item
	}

	return lastItem.Match(lex.Separator, lex.EOF)
}

// balanced checks if the string has balanced {} and (). Does not correctly
// handle quoted strings.
func balanced(s string) bool {
	if count(s, '{') != count(s, '}') {
		return false
	}

	if count(s, '(') != count(s, ')') {
		return false
	}

	return true
}

func count(s string, r rune) int {
	c := 0
	for _, x := range s {
		if x == r {
			c++
		}
	}
	return c
}

func runFile(name string, input io.Reader) error {

	ctx := compile.NewTopContext()

	return runProgram(ctx, name, input, false)
}

func runProgram(ctx *compile.Context, name string, input io.Reader, printResult bool) error {

	p := parser.NewFromReader(name, input)

	parsed := p.Parse()
	// log.Printf("parsed: %s", parsed)

	program, err := compile.Compile(parsed)
	if err != nil {
		return err
	}

	result, err := program(ctx)
	if err != nil {
		return err
	}

	if printResult {
		fmt.Println(result.(compile.Tuple).Values[1])
	}

	return nil
}
