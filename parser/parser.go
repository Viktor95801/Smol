package parser

import (
	. "smol/lexer/token"
	. "smol/parser/environment"
	"strconv"

	"fmt"
	"strings"
)

const (
	ERROR_MESSAGE_FORMAT = "%s:%d:%d: ERROR: %s"
	NOTE_MESSAGE_FORMAT  = "%s:%d:%d: NOTE: %s"
)

type Parser struct {
	file             *string
	input            []Token
	source           []string
	pos              int
	ptok, ctok, ntok Token

	Errors []error
	Notes  []string
}

func NewParser(file string, source *string, input []Token) *Parser {
	parser := &Parser{}
	parser.Init(&file, source, input)
	return parser
}

func (p *Parser) Init(file *string, source *string, input []Token) {
	p.file = file
	p.input = input
	p.source = strings.Split(*source, "\n")
	p.pos = -2

	p.next()
	p.next()
}

// TODO: better error messages
// TODO: better backtracking. Currently: none
func (p *Parser) Parse() (tree Node, ok bool) {
	return p.program()
}

func (p *Parser) next() {
	p.ptok = p.ctok
	p.ctok = p.ntok

	p.pos++
	if p.pos+1 < len(p.input) {
		p.ntok = p.input[p.pos+1]
	} else {
		p.ntok = Token{Kind: TokEOF}
	}
}

func (p *Parser) consume(kinds ...TokenKind) bool {
	if p.ctok.OneOf(kinds...) {
		p.next()
		return true
	}
	return false
}

func (p *Parser) panic() {
	for !p.consume(TokSemi, TokEOF) {
		p.next()
	}
}

func (p *Parser) errorMsg(tok Token, format string, a ...any) error {
	return fmt.Errorf(
		ERROR_MESSAGE_FORMAT,
		*p.file, tok.Pos.Line, tok.Pos.Col, fmt.Sprintf(format, a...),
	)
}

func (p *Parser) errorAt(tok Token, format string, a ...any) {
	p.Errors = append(p.Errors, p.errorMsg(tok, format, a...))
	p.panic()
}

func (p *Parser) error(format string, a ...any) {
	p.errorAt(p.ptok, format, a...)
}

func (p *Parser) noteMsg(tok Token, format string, a ...any) string {
	return fmt.Sprintf(
		NOTE_MESSAGE_FORMAT,
		*p.file, tok.Pos.Line, tok.Pos.Col, fmt.Sprintf(format, a...),
	)
}

func (p *Parser) noteAt(tok Token, format string, a ...any) {
	p.Notes = append(p.Notes, p.noteMsg(tok, format, a...))
	p.panic()
}

func (p *Parser) note(format string, a ...any) {
	p.noteAt(p.ptok, format, a...)
}

func (p *Parser) program() (Statement, bool) {
	program := Program{}
	program.Start = p.ptok.Pos

	had_error := false
	for !p.consume(TokEOF) {
		stmt, ok := p.statement()
		if !ok {
			had_error = true
			continue
		} else if _, ok := stmt.(*StmtEmpty); ok {
			continue
		}
		program.Children = append(program.Children, stmt)
	}
	program.End = p.ptok.Pos

	return &program, !had_error
}

func (p *Parser) statement() (Statement, bool) {
	if p.ctok.OneOf(TokEOF) {
		return nil, false
	}

	if p.consume(TokSemi) {
		return &StmtEmpty{Where: p.ptok.Pos}, true
	}

	if p.consume(TokLCurly) {
		result, ok := p.stmt_block()
		return result, ok
	}

	if p.consume(KwPrint, KwPrintln) {
		is_println := false
		if p.ptok.OneOf(KwPrintln) {
			is_println = true
		}

		start := p.ptok.Pos

		if p.consume(TokSemi) {
			if !is_println {
				p.note("Empty 'print' statement doesn't do anything.")
			}
			return &StmtPrint{IsPrintln: is_println, Start: start, End: p.ptok.Pos, Values: nil}, true
		}
		values := make([]Expression, 1)
		expr, ok := p.expression()
		if !ok {
			return nil, false
		}
		values[0] = expr
		for p.consume(TokComma) {
			expr, ok = p.expression()
			if !ok {
				return nil, false
			}
			values = append(values, expr)
		}
		if !p.consume(TokSemi) {
			p.error("Expected ';'.")
			return nil, false
		}
		return &StmtPrint{IsPrintln: is_println, Start: start, End: p.ptok.Pos, Values: values}, true
	}

	if p.consume(KwReturn) {
		start := p.ptok.Pos

		expr, ok := p.expression()
		if !ok {
			return nil, false
		}
		if !p.consume(TokSemi) {
			p.error("Expected ';'.")
			return nil, false
		}
		return &StmtReturn{Start: start, End: p.ptok.Pos, Expr: expr}, true
	}

	if p.consume(KwLet) {
		return p.stmt_assign()
	}

	if p.consume(KwIf) {
		return p.stmt_if()
	}

	start := p.ptok.Pos
	expr, ok := p.expression()
	if !ok {
		return nil, false
	}

	if !p.consume(TokSemi) {
		p.error("Expected ';'.")
		return nil, false
	}

	return &StmtExpression{
		Start: start,
		End:   p.ptok.Pos,
		Expr:  expr,
	}, true
}

func (p *Parser) stmt_block() (Statement, bool) {
	block := StmtBlock{}
	block.Start = p.ptok.Pos

	had_error := false
	for !p.consume(TokEOF, TokRCurly) {
		stmt, ok := p.statement()
		if !ok {
			had_error = true
			continue
		} else if _, ok := stmt.(*StmtEmpty); ok {
			continue
		}
		block.Children = append(block.Children, stmt)
	}
	block.End = p.ptok.Pos
	if p.ptok.OneOf(TokEOF) {
		p.error("Expected '}' (unclosed block).")
		had_error = true
	}

	return &block, !had_error
}

func (p *Parser) stmt_assign() (Statement, bool) {
	if !p.consume(TokIdent) {
		p.error("Expected identifier.")
		return nil, false
	}
	name := p.ptok

	if !p.consume(TokColon) {
		p.error("Expected ':'.")
		return nil, false
	}

	var type_expression *TypeExpression
	if !p.ctok.OneOf(TokAss, TokColon) {
		var ok bool
		type_expression, ok = p.type_expression()
		if !ok {
			return nil, false
		}
	}

	is_const := false
	if p.consume(TokColon) {
		is_const = true
		/* } else if p.consume(TokSemi) {
		variable := &ExprVariable{
			Where: name.Pos,
			Const: is_const,
			Name:  name.Literal,
		}
		return &StmtAssign{
			Start:    name.Pos,
			End:      p.ptok.Pos,
			Var:      variable,
			TypeExpr: type_expression,
			Value:    nil,
			}, true */
	} else if !p.consume(TokAss) {
		p.error("Expected '='.")
		return nil, false
	}

	value, ok := p.expression()
	if !ok {
		return nil, false
	}

	if !p.consume(TokSemi) {
		p.error("Expected ';'.")
		return nil, false
	}

	variable := &ExprVariable{
		Where: name.Pos,
		Const: is_const,
		Name:  name.Literal,
	}
	return &StmtAssign{
		Start:    name.Pos,
		End:      p.ptok.Pos,
		Var:      variable,
		TypeExpr: type_expression,
		Value:    value,
	}, true
}

func (p *Parser) stmt_if() (Statement, bool) {
	start := p.ptok.Pos
	cond, ok := p.expression()
	if !ok {
		return nil, false
	}

	if !p.consume(TokLCurly) {
		p.error("Expected '{'.")
		return nil, false
	}

	if_block, ok := p.stmt_block()
	if !ok {
		return nil, false
	}

	if !p.consume(KwElse) {
		return &StmtIf{
			Start:     start,
			End:       p.ptok.Pos,
			Condition: cond,
			If:        if_block.(*StmtBlock),
			Else:      nil,
		}, true
	}

	if p.ctok.OneOf(KwIf) {
		else_block, ok := p.statement()
		if !ok {
			return nil, false
		}

		return &StmtIf{
			Start:     start,
			End:       p.ptok.Pos,
			Condition: cond,
			If:        if_block,
			Else:      else_block,
		}, true
	} else if !p.consume(TokLCurly) {
		p.error("Expected '{'.")
		return nil, false
	}

	else_block, ok := p.stmt_block()
	if !ok {
		return nil, false
	}

	return &StmtIf{
		Start:     start,
		End:       p.ptok.Pos,
		Condition: cond,
		If:        if_block,
		Else:      else_block,
	}, true
}

func (p *Parser) expression() (Expression, bool) {
	return p.assignment()
}

func (p *Parser) assignment() (Expression, bool) {
	left, ok := p.term()
	if !ok {
		return nil, false
	}

	if p.ctok.OneOf(TokAss) {
		error_token := p.ptok
		p.next()

		variable, is_var := left.(*ExprVariable)
		if !is_var {
			p.error("The left side of an assignment must be a valid identifier.")
			return nil, false
		}

		if variable.Const {
			p.errorAt(error_token, "Assigning to constant variable '%s'.", variable.Name)
			return nil, false
		}

		right, ok := p.expression()
		if !ok {
			return nil, false
		}

		return &ExprAssign{
			Var:   variable,
			Value: right,
		}, true
	}

	return left, true
}

func (p *Parser) term() (Expression, bool) {
	left, ok := p.factor()
	if !ok {
		return nil, false
	}

	for p.ctok.OneOf(OpAdd, OpSub) {
		op := p.ctok
		p.next()
		right, ok := p.factor()
		if !ok {
			return nil, false
		}

		left = &ExprBinary{
			Op:    op,
			Left:  left,
			Right: right,
		}
	}

	return left, true
}

func (p *Parser) factor() (Expression, bool) {
	left, ok := p.power()
	if !ok {
		return nil, false
	}

	for p.ctok.OneOf(OpMul, OpDiv, OpMod) {
		op := p.ctok
		p.next()
		right, ok := p.power()
		if !ok {
			return nil, false
		}

		left = &ExprBinary{
			Op:    op,
			Left:  left,
			Right: right,
		}
	}

	return left, true
}

func (p *Parser) power() (Expression, bool) {
	left, ok := p.unary()
	if !ok {
		return nil, false
	}

	if p.ctok.OneOf(OpPow) {
		op := p.ctok
		p.next()
		right, ok := p.power()
		if !ok {
			return nil, false
		}

		left = &ExprBinary{
			Op:    op,
			Left:  left,
			Right: right,
		}
	}

	return left, true
}

func (p *Parser) unary() (Expression, bool) {
	if !p.ctok.OneOf(OpAdd, OpSub) {
		start := p.ctok.Pos
		value, ok := p.primary()
		if !ok {
			return nil, false
		}

		if p.consume(TokLBrack) {
			index, ok := p.expression()
			if !ok {
				return nil, false
			}
			if !p.consume(TokRBrack) {
				p.error("Expected ']'.")
				return nil, false
			}

			return &ExprArrayAccess{
				Start: start, End: p.ptok.Pos,
				Array: value, Index: index,
			}, true
		} else {
			return value, true
		}
	}

	op := p.ctok
	p.next()
	value, ok := p.unary()
	if !ok {
		return nil, false
	}

	return &ExprUnary{
		Start: op.Pos, End: p.ptok.Pos,
		Op:    op,
		Value: value,
	}, true
}

func (p *Parser) primary() (Expression, bool) {
	if p.consume(TokLParen) {
		start := p.ptok.Pos
		insides, ok := p.expression()
		if !ok {
			return nil, false
		}
		if !p.consume(TokRParen) {
			p.error("Expected ')'.")
			return nil, false
		}

		return &ExprGroup{
			Start: start,
			End:   p.ptok.Pos,
			Value: insides,
		}, true
	}

	p.next()
	result := &ExprLiteral{
		Where: p.ptok.Pos,
	}
	switch p.ptok.Kind {
	case TokNumber:
		value, err := strconv.ParseFloat(p.ptok.Literal, 64)
		if err != nil {
			p.error("Invalid number.")
			return nil, false
		}
		result.Value = ValueNumber(value)
	case TokString:
		result.Value = ValueString(p.ptok.Literal)
	case TokIdent:
		return &ExprVariable{
			Where: p.ptok.Pos,
			Const: false,
			Name:  p.ptok.Literal,
		}, true
	case KwTrue, KwFalse:
		result.Value = ValueBool(p.ptok.Kind == KwTrue)

	case TokLBrack:
		start := p.ptok.Pos

		first, ok := p.expression()
		if !ok {
			return nil, false
		}
		values := []Expression{first}
		for p.consume(TokComma) {
			field, ok := p.expression()
			if !ok {
				return nil, false
			}
			values = append(values, field)
		}

		if !p.consume(TokRBrack) {
			println(p.ptok.String(), p.ctok.String(), p.ntok.String())
			p.error("Expected ']'.")
			return nil, false
		}

		return &ExprArray{
			Start:  start,
			End:    p.ptok.Pos,
			Fields: values,
		}, true

	default:
		p.error("Expected a literal.")
		return nil, false
	}
	return result, true
}

func (p *Parser) type_expression() (*TypeExpression, bool) {
	println("A")
	if !p.consume(TokIdent) {
		println(p.ptok.String(), p.ctok.String(), p.ntok.String())
		p.error("Expected type name.")
		return nil, false
	}
	name := p.ptok

	var fields []*TypeExpression = nil
	if p.consume(TokLBrack) {
		for !p.consume(TokRBrack, TokEOF) {
			typ, ok := p.type_expression()
			if !ok {
				return nil, false
			}
			fields = append(fields, typ)
			if !p.consume(TokComma) && !p.ctok.OneOf(TokRBrack, TokEOF) {
				p.error("Expected ','.")
				return nil, false
			}
		}
		if !p.ptok.OneOf(TokRBrack) {
			p.error("Expected ']'.")
			return nil, false
		}

		return &TypeExpression{
			Start:   name.Pos,
			End:     p.ptok.Pos,
			Name:    name.Literal,
			Fields:  fields,
			Returns: nil,
		}, true
	}

	var returns []*TypeExpression = nil
	if p.consume(TokLBrack) {
		for !p.consume(TokRBrack, TokEOF) {
			typ, ok := p.type_expression()
			if !ok {
				return nil, false
			}
			returns = append(returns, typ)
			if !p.consume(TokComma) && !p.ntok.OneOf(TokRBrack, TokEOF) {
				p.error("Expected ','.")
				return nil, false
			}
		}
		if !p.ptok.OneOf(TokRBrack) {
			p.error("Expected ']'.")
			return nil, false
		}

		return &TypeExpression{
			Start:   name.Pos,
			End:     p.ptok.Pos,
			Name:    name.Literal,
			Fields:  fields,
			Returns: returns,
		}, true
	}

	return &TypeExpression{
		Start:   name.Pos,
		End:     p.ptok.Pos,
		Name:    name.Literal,
		Fields:  nil,
		Returns: nil,
	}, true
}
