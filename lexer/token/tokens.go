package token

import "slices"

type TokenPosition struct {
	Line int
	Col  int
	End  int
}

type Token struct {
	Kind    TokenKind
	Literal string
	Pos     TokenPosition
}

func (t Token) String() string {
	if t.Kind == TokString {
		return "\"" + t.Literal + "\""
	}
	return t.Literal
}

func (t Token) OneOf(kinds ...TokenKind) bool {
	return slices.Contains(kinds, t.Kind)
}

func (t TokenKind) String() string {
	return kindNames[t]
}

type TokenKind uint16

const (
	TokEOF TokenKind = iota
	TokInvalid
	TokIdent
	TokNumber
	TokString

	TokSemi
	TokComma
	TokLParen
	TokRParen
	TokLCurly
	TokRCurly
	TokLBrack
	TokRBrack

	TokAss
	TokColon

	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpPow

	KwTrue
	KwFalse

	KwReturn
	KwPrint
	KwPrintln
	KwLet
	KwIf
	KwElse
)

var kindNames = [...]string{
	TokEOF:    "EOF",
	TokIdent:  "Ident",
	TokNumber: "Number",
	TokString: "String",
	TokComma:  "Comma",
	TokSemi:   "Semi",
	TokLParen: "LParen",
	TokRParen: "RParen",
	TokLCurly: "LCurly",
	TokRCurly: "RCurly",
	TokLBrack: "LBrack",
	TokRBrack: "RBrack",
	OpAdd:     "OpAdd",
	OpSub:     "OpSub",
	OpMul:     "OpMul",
	OpDiv:     "OpDiv",
	OpMod:     "OpMod",
	OpPow:     "OpPow",
	KwTrue:    "KwTrue",
	KwFalse:   "KwFalse",
	KwReturn:  "KwReturn",
	KwPrint:   "KwPrint",
	KwPrintln: "KwPrintln",
	KwIf:      "KwIf",
	KwElse:    "KwElse",
	KwLet:     "KwLet",
	TokAss:    "TokAss",
	TokColon:  "TokColon",
}
