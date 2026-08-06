package lexer

import (
	. "smol/lexer/token"
	. "smol/util"

	"fmt"
	"strconv"
)

type Lexer struct {
	file           *string
	input          string
	tokens         []Token
	bol, pos, line int
	c              uint8

	Errors []error
}

func NewLexer(file string, input string) *Lexer {
	l := &Lexer{}
	l.Init(&file, input)
	return l
}

func (l *Lexer) Init(file *string, input string) {
	l.file = file
	l.input = input
	l.tokens = make([]Token, 0)
	l.bol = 0
	l.pos = -1
	l.line = 1

	l.next()
}

var singleCharTokMap = [...]TokenKind{
	'+': OpAdd,
	'-': OpSub,
	'*': OpMul,
	'/': OpDiv,
	'%': OpMod,
	'&': OpBitAnd,
	'|': OpBitOr,
	'^': OpBitXor,
	'~': OpBitNot,
	'>': OpGt,
	'<': OpLt,
	'!': OpNot,
	',': TokComma,
	';': TokSemi,
	'(': TokLParen,
	')': TokRParen,
	'{': TokLCurly,
	'}': TokRCurly,
	'[': TokLBrack,
	']': TokRBrack,
	'=': TokAss,
	':': TokColon,
}

var secondCharTokMap = [...]struct {
	c    byte
	kind TokenKind
}{
	OpMul:    {'*', OpPow},
	OpGt:     {'=', OpGe},
	OpLt:     {'=', OpLe},
	TokAss:   {'=', OpEq},
	OpNot:    {'=', OpNe},
	OpBitAnd: {'&', OpAnd},
	OpBitOr:  {'|', OpOr},
}

func (l *Lexer) Lex() ([]Token, bool) {
	had_error := false

	for l.c != 0 {
		if IsSpace(l.c) {
			l.next()
			continue
		}

		if IsAlpha(l.c) {
			not_ok := !l.lexIdent()
			had_error = had_error || not_ok
			continue
		}

		if IsDigit(l.c) {
			not_ok := !l.lexNumber() // for some reason, the Go compiler glitches the fuck out and doesnt run this function if I put it in directly
			had_error = had_error || not_ok
			continue
		}

		if l.c == '"' {
			not_ok := !l.lexString()
			had_error = had_error || not_ok
			continue
		}

		kind := singleCharTokMap[l.c]
		//TODO: Ugly AF code, refactor later
		if kind != 0 {
			if int(kind) < len(secondCharTokMap) {
				if next_char := secondCharTokMap[kind]; next_char.c != 0 && l.pos+1 < len(l.input) && l.input[l.pos+1] == next_char.c {
					l.tokens = append(l.tokens, l.newTok(next_char.kind, string(l.c)+string(next_char.c)))
					l.next()
				} else {
					l.tokens = append(l.tokens, l.newTok(kind, string(l.c)))
				}
			} else {
				l.tokens = append(l.tokens, l.newTok(kind, string(l.c)))
			}
		} else {
			tok := l.newTok(TokInvalid, string(l.c))
			had_error = true
			l.Errors = append(l.Errors, fmt.Errorf("Invalid token on stream: %s", tok))
			l.tokens = append(l.tokens, tok)
		}
		l.next()
	}

	l.tokens = append(l.tokens, l.newTok(TokEOF, "EOF"))
	return l.tokens, !had_error
}

func (l *Lexer) newTok(kind TokenKind, lit string) Token {
	return Token{
		Kind:    kind,
		Literal: lit,
		Pos: TokenPosition{
			Line: l.line,
			Col:  l.pos - l.bol,
			End:  l.pos - l.bol + len(lit),
		},
	}
}

func (l *Lexer) next() {
	l.pos++
	if l.pos < len(l.input) {
		l.c = l.input[l.pos]
		if l.c == '\n' {
			l.line++
			l.bol = l.pos
		}
	} else {
		l.c = 0
	}
}

var keywordMap = map[string]TokenKind{
	"true":    KwTrue,
	"false":   KwFalse,
	"return":  KwReturn,
	"print":   KwPrint,
	"println": KwPrintln,
	"input":   KwInput,
	"let":     KwLet,
	"if":      KwIf,
	"while":   KwWhile,
	"else":    KwElse,
	"type":    KwType,
	"struct":  KwStruct,
}

func (l *Lexer) lexIdent() bool {
	pos := l.pos
	for IsAlnum(l.c) {
		l.next()
	}

	lit := l.input[pos:l.pos]
	tok := l.newTok(TokIdent, lit)
	if kind, ok := keywordMap[lit]; ok {
		tok.Kind = kind
		l.tokens = append(l.tokens, tok)
	} else {
		l.tokens = append(l.tokens, tok)
	}
	return true
}

func (l *Lexer) lexNumber() bool {
	pos := l.pos
	dot_count := 0
	contains_letters := false
	for IsAlnum(l.c) || l.c == '.' {
		if l.c == '.' {
			dot_count++
		}
		if IsAlpha(l.c) {
			contains_letters = true
		}
		l.next()
	}

	lit := l.input[pos:l.pos]
	tok := l.newTok(TokNumber, lit)
	if dot_count > 1 {
		l.Errors = append(l.Errors, fmt.Errorf("Invalid number (too many dots): %s", lit))

		tok.Kind = TokInvalid
		l.tokens = append(l.tokens, tok)
		return false
	}
	if contains_letters {
		l.Errors = append(l.Errors, fmt.Errorf("Invalid number (contains letters): %s", lit))

		tok.Kind = TokInvalid
		l.tokens = append(l.tokens, tok)
		return false
	}

	l.tokens = append(l.tokens, tok)
	return true
}

func (l *Lexer) lexString() bool {
	pos := l.pos
	l.next()
	for l.c != '"' {
		if l.c == 0 {
			l.Errors = append(l.Errors, fmt.Errorf("Unterminated string: %s", l.input[pos:]))
			return false
		}
		if l.c == '\\' {
			l.next()
		}
		l.next()
	}
	l.next()

	lit := l.input[pos:l.pos]
	tok := l.newTok(TokString, lit)
	new_lit, err := strconv.Unquote(lit)
	if err != nil {
		l.Errors = append(l.Errors, fmt.Errorf("Invalid string: %s. %s", lit, err))
		tok.Kind = TokInvalid
		l.tokens = append(l.tokens, tok)
		return false
	}
	tok.Literal = new_lit
	l.tokens = append(l.tokens, tok)
	return true
}
