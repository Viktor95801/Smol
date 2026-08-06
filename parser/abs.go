package parser

import (
	. "smol/lexer/token"
	. "smol/parser/environment"
	"strings"
)

type Node interface {
	String() string
	// Returns the start and end of the node for error output reasons
	Pos() (TokenPosition, TokenPosition) //TODO: use
}

type Statement Node

type Program struct {
	IsModule bool // UNUSED rn
	StmtBlock
}

func (s *Program) String() string {
	return "Program" + s._String("")
}

type StmtLoopBlock struct {
	StmtBlock
}

type StmtBlock struct {
	Start, End TokenPosition
	Children   []Statement
}

func (s *StmtBlock) _String(indent string) string {
	sb := strings.Builder{}
	for _, child := range s.Children {
		if block, is_block := child.(*StmtBlock); is_block {
			sb.WriteString(block._String("  " + indent))
			sb.WriteByte('\n')
			continue
		}
		sb.WriteString(indent)
		sb.WriteByte(' ')
		sb.WriteString(child.String())
		sb.WriteByte('\n')
	}
	result := indent + "[\n" + sb.String() + indent + "]"
	return result
}

func (s *StmtBlock) String() string {
	return s._String("")
}

func (s *StmtBlock) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type StmtBreak struct {
	Where TokenPosition
}

func (s *StmtBreak) String() string {
	return "[break]"
}

func (s *StmtBreak) Pos() (TokenPosition, TokenPosition) {
	return s.Where, s.Where
}

type StmtContinue struct {
	Where TokenPosition
}

func (s *StmtContinue) String() string {
	return "[continue]"
}

func (s *StmtContinue) Pos() (TokenPosition, TokenPosition) {
	return s.Where, s.Where
}

type StmtEmpty struct {
	Where TokenPosition
}

func (*StmtEmpty) String() string {
	return "[ ]"
}

func (s *StmtEmpty) Pos() (TokenPosition, TokenPosition) {
	return s.Where, s.Where
}

type StmtExpression struct {
	Start, End TokenPosition
	Expr       Expression
}

func (s *StmtExpression) String() string {
	return "[" + s.Expr.String() + "]"
}

func (s *StmtExpression) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type StmtPrint struct {
	IsPrintln  bool
	Start, End TokenPosition
	Values     []Expression
}

func (s *StmtPrint) String() string {
	if len(s.Values) == 1 {
		return "[print " + s.Values[0].String() + "]"
	}
	sb := strings.Builder{}
	sb.WriteString("[print ")
	for i, value := range s.Values {
		sb.WriteString(value.String())
		if i < len(s.Values) {
			sb.WriteString(", ")
		}
	}
	sb.WriteByte(']')
	return sb.String()
}

func (s *StmtPrint) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type StmtReturn struct {
	Start, End TokenPosition
	Expr       Expression
}

func (s *StmtReturn) String() string {
	return "[return " + s.Expr.String() + "]"
}

func (s *StmtReturn) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type StmtAssign struct {
	Start, End TokenPosition
	Var        *ExprVariable
	TypeExpr   *ExprTypeRef
	Value      Expression
}

func (s *StmtAssign) String() string {
	assign := "="
	if s.Var.IsConst {
		assign = ":"
	}
	typ_str := ""
	if s.TypeExpr != nil {
		typ_str = s.TypeExpr.String()
	}

	if s.Value == nil {
		return "[var " + s.Var.String() + ":" + typ_str + "]"
	} else {
		val_str := s.Value.String()
		return "[var " + s.Var.String() + ":" + typ_str + assign + val_str + "]"
	}
}

func (s *StmtAssign) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type StmtIf struct {
	Start, End TokenPosition
	Condition  Expression

	If   Statement
	Else Statement
}

func (s *StmtIf) String() string {
	else_part := ""
	if s.Else != nil {
		else_part = " else " + s.Else.String()
	}
	return "[if " + s.Condition.String() + " " + s.If.String() + else_part + "]"
}

func (s *StmtIf) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type StmtWhile struct {
	Start, End TokenPosition
	Condition  Expression
	Body       Statement
}

func (s *StmtWhile) String() string {
	return "[while " + s.Condition.String() + " " + s.Body.String() + "]"
}

func (s *StmtWhile) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type StmtDeclType struct {
	Start, End TokenPosition
	Name       string
	TypeExpr   *ExprTypeRef
}

func (s *StmtDeclType) String() string {
	return "[type " + s.Name + " " + s.TypeExpr.String() + "]"
}

func (s *StmtDeclType) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type Expression Node

type ExprBinary struct {
	Start, End TokenPosition
	Left       Expression
	Right      Expression
	Op         Token
}

func (e *ExprBinary) String() string {
	return "(" + e.Left.String() + " " + e.Op.String() + " " + e.Right.String() + ")"
}

func (s *ExprBinary) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type ExprUnary struct {
	Start, End TokenPosition
	Op         Token
	Value      Expression
}

func (e *ExprUnary) String() string {
	return "(" + e.Op.String() + "" + e.Value.String() + ")"
}

func (s *ExprUnary) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type ExprArrayAccess struct {
	Start, End TokenPosition
	Array      Expression
	Index      Expression
}

func (e *ExprArrayAccess) String() string {
	return "(" + e.Array.String() + "[" + e.Index.String() + "])"
}

func (s *ExprArrayAccess) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type ExprLiteral struct {
	Where TokenPosition
	Value Value
}

func (e *ExprLiteral) String() string {
	return e.Value.String()
}

func (s *ExprLiteral) Pos() (TokenPosition, TokenPosition) {
	return s.Where, s.Where
}

type ExprAssign struct {
	Start, End TokenPosition
	Var        *ExprVariable
	Value      Expression
}

func (e *ExprAssign) String() string {
	return "(" + e.Var.Name + " = " + e.Value.String() + ")"
}

func (s *ExprAssign) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type ExprVariable struct {
	Where   TokenPosition
	IsConst bool
	Name    string
}

func (e *ExprVariable) String() string {
	return e.Name
}

func (s *ExprVariable) Pos() (TokenPosition, TokenPosition) {
	return s.Where, s.Where
}

type ExprGroup struct {
	Start, End TokenPosition
	Value      Expression
}

func (e *ExprGroup) String() string {
	return e.Value.String()
}

func (s *ExprGroup) Pos() (TokenPosition, TokenPosition) {
	return s.Start, s.End
}

type ExprArray struct {
	Start, End TokenPosition
	Fields     []Expression
}

func (e *ExprArray) String() string {
	sb := strings.Builder{}
	sb.WriteByte('[')
	for i, field := range e.Fields {
		sb.WriteString(field.String())
		if i < len(e.Fields)-1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')
	return sb.String()
}

func (e *ExprArray) Pos() (TokenPosition, TokenPosition) {
	return e.Start, e.End
}

type ExprTypeStruct struct {
	Start, End    TokenPosition
	IsDeclaration bool
	Name          string
	Fields        []StmtAssign
}

func (t *ExprTypeStruct) String() string {
	if len(t.Fields) == 0 {
		return t.Name
	}

	sb := strings.Builder{}
	sb.WriteString(t.Name)
	if len(t.Fields) > 0 {
		sb.WriteByte('{')
		for _, field := range t.Fields {
			sb.WriteString(field.String())
			sb.WriteByte(';')
		}
		sb.WriteByte('}')
	}

	return sb.String()
}

func (t *ExprTypeStruct) Pos() (TokenPosition, TokenPosition) {
	return t.Start, t.End
}

// Basic ExprTypeRef construct for variable declarations and type declarations. E.g.: a: ExprTypeRef = Some Value...
type ExprTypeRef struct {
	Start, End TokenPosition
	Name       string
	Fields     []*ExprTypeRef
	Returns    []*ExprTypeRef // specifically for function types
}

func (t *ExprTypeRef) String() string {
	if len(t.Fields) == 0 && len(t.Returns) == 0 {
		return t.Name
	}

	sb := strings.Builder{}
	sb.WriteString(t.Name)
	if len(t.Fields) > 0 {
		sb.WriteByte('[')
		for _, field := range t.Fields {
			sb.WriteString(field.String())
			sb.WriteByte(',')
		}
		sb.WriteByte(']')
	}

	if len(t.Returns) > 0 {
		if len(t.Fields) == 0 {
			sb.WriteString("[]")
		}
		sb.WriteByte('[')
		for _, ret := range t.Returns {
			sb.WriteString(ret.String())
			sb.WriteByte(',')
		}
		sb.WriteByte(']')
	}

	return sb.String()
}

func (t *ExprTypeRef) Pos() (TokenPosition, TokenPosition) {
	return t.Start, t.End
}
