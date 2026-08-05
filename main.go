package main

import (
	"smol/checkers"
	"smol/lexer"
	"smol/parser"
	. "smol/parser/environment"

	"fmt"
	"os"
)

func init_environment(env *ValueScope) {
	env.NewConstant("Pi", ValueNumber(3.14))
}

func init_type_scope(ts *TypeScope) {
	ts.Set(NumberType.Name(), NumberType)
	ts.Set(BoolType.Name(), BoolType)
	ts.Set(StringType.Name(), StringType)
	ts.Set(ArrayType.Name(), ArrayType)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <filename>")
		os.Exit(1)
	}
	file := os.Args[1]
	_content, err := os.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	content := string(_content)

	l := lexer.NewLexer(file, content)

	tokens, ok := l.Lex()
	//for _, tok := range tokens {
	//	fmt.Printf("%s:%d:%d %s: '%s'\n", file, tok.Pos.Line, tok.Pos.Col, tok.Kind, tok)
	//}
	if !ok {
		fmt.Println("LEXER ERRORS:")
		for _, err := range l.Errors {
			fmt.Println(" ", err)
		}
		os.Exit(1)
	}

	p := parser.NewParser(file, &content, tokens)

	tree, ok := p.Parse()
	if !ok {
		fmt.Println("PARSER ERRORS:")
		for _, err := range p.Errors {
			fmt.Println(" ", err)
		}
		os.Exit(1)
	}
	fmt.Println(tree)

	code, ok := tree.(*parser.Program)
	if !ok {
		fmt.Println("UNREACHABLE")
		os.Exit(1)
	}

	env := NewValueScope(nil)
	init_environment(env)
	ts := NewTypeScope(nil)
	init_type_scope(ts)

	c := checkers.NewChecker(file, env, ts)
	if !c.Check(code) {
		fmt.Println("CHECKER ERRORS:")
		for _, err := range c.Errors {
			fmt.Println(" ", err)
		}
		os.Exit(1)
	}

	env = NewValueScope(nil)
	init_environment(env)
	ts = NewTypeScope(nil)
	init_type_scope(ts)

	error_code, err := parser.Interpret(code, env, ts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("exit status:", error_code)

	//result, err := parser.Evaluate(tree)
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//fmt.Println(result)
}
