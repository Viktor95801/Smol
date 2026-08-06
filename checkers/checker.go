package checkers

import (
	p "smol/parser"
	. "smol/parser/environment"
	. "smol/util"

	//	. "smol/lexer/token"

	"fmt"
)

type Checker struct {
	Errors []error

	file *string
	vs   *ValueScope
	ts   *TypeScope
}

func NewChecker(file string, vs *ValueScope, ts *TypeScope) *Checker {
	c := &Checker{}
	c.Init(&file, vs, ts)
	return c
}

func (c *Checker) Init(file *string, vs *ValueScope, ts *TypeScope) {
	c.file = file
	c.vs = vs
	c.ts = ts
	c.Errors = make([]error, 0)
}

func (c *Checker) errorMsg(n p.Node, format string, a ...any) error {
	pos, _ := n.Pos()
	return fmt.Errorf(
		p.ERROR_MESSAGE_FORMAT,
		*c.file, pos.Line, pos.Col, fmt.Sprintf(format, a...),
	)
}

func (c *Checker) errorMsg_End(n p.Node, format string, a ...any) error {
	_, pos := n.Pos()
	return fmt.Errorf(
		p.ERROR_MESSAGE_FORMAT,
		*c.file, pos.Line, pos.End, fmt.Sprintf(format, a...),
	)
}

func (c *Checker) error(n p.Node, format string, a ...any) {
	c.Errors = append(c.Errors, c.errorMsg(n, format, a...))
}

func (c *Checker) errorEnd(n p.Node, format string, a ...any) {
	c.Errors = append(c.Errors, c.errorMsg_End(n, format, a...))
}

func (c *Checker) Check(code p.Node) bool {
	if code == nil {
		panic("code is nil")
	}
	c.checkStmt(code)

	return len(c.Errors) == 0
}

func (c *Checker) checkStmt(code p.Node) {
	switch n := code.(type) {
	case *p.Program:
		for _, child := range n.Children {
			c.Check(child)
		}
	case *p.StmtBlock:
		parent_vs := c.vs
		parent_ts_types := c.ts

		c.vs = NewValueScope(parent_vs)
		c.ts = NewTypeScope(parent_ts_types)
		for _, child := range n.Children {
			c.Check(child)
		}
		c.vs = parent_vs
		c.ts = parent_ts_types
	case *p.StmtAssign:
		if c.vs.Exists(n.Var.Name) {
			c.errorEnd(n.Var, "Variable '%s' already exists", n.Var.Name)
			break
		}

		var typ Type
		if n.TypeExpr != nil {
			typ = c.checkTypeExpr(n.Var, *n.TypeExpr)
			if typ == nil {
				return
			}
		}

		if n.Value != nil {
			value := c.checkExpr(n.Value)
			if value == nil {
				return
			}
			if typ != nil && !value.IsAssignableTo(typ) {
				println(value.String())
				println(typ.String())
				c.error(n.Var, "Cannot assign '%s' to '%s'.", value.Name(), typ.Name())
				return
			}
			typ = value
		}

		c.vs.Mark(n.Var.Name, n.Var.IsConst, typ)
	case *p.StmtExpression:
		c.checkExpr(n.Expr)
	case *p.StmtIf:
		cond := c.checkExpr(n.Condition)
		if cond == nil {
			return
		}
		if !Is[TypeBool](cond) {
			c.error(n.Condition, "Expected bool, got '%s'.", cond.Name())
		}
		c.checkStmt(n.If)
		if n.Else != nil {
			c.checkStmt(n.Else)
		}
	case *p.StmtReturn:
		expr := c.checkExpr(n.Expr)
		if expr == nil {
			return
		}

		if !Is[TypeNumber](expr) {
			c.error(n.Expr, "Expected number, got '%s'.", expr.Name())
		}
	case *p.StmtPrint:
		for _, value := range n.Values {
			c.checkExpr(value)
		}
	case *p.StmtDeclType:
		typ := c.checkTypeExpr(n.TypeExpr, *n.TypeExpr)
		if typ == nil {
			return
		}
		c.ts.Set(n.Name, typ)
	default:
		panic(fmt.Sprintf("UNHANDLED CASE ON checkStmt(%s)", code))
	}
}

func (c *Checker) checkExpr(code p.Expression) Type {
	if code == nil {
		panic(fmt.Sprintf("NULL code AT checkExpr: %s", code))
	}

	switch n := code.(type) {
	case *p.ExprGroup:
		return c.checkExpr(n.Value)
	case *p.ExprArray:
		main_type := c.checkExpr(n.Fields[0])
		if main_type == nil {
			return nil
		}

		for _, field := range n.Fields[1:] {
			if !main_type.IsAssignableTo(c.checkExpr(field)) {
				c.error(field, "Mixing types in array ('%s' vs '%s').", main_type.Name(), c.checkExpr(field).Name())
			}
		}

		return &TypeArray{ElemType: main_type}
	case *p.ExprVariable:
		if !c.vs.Exists(n.Name) {
			c.error(n, "Undefined variable '%s'.", n.Name)
			return nil
		}
		n.IsConst = c.vs.IsConst(n.Name)
		return c.vs.Type(n.Name)
	case *p.ExprAssign:
		if !c.vs.Exists(n.Var.Name) {
			c.error(n.Var, "Undefined variable '%s'.", n.Var.Name)
			return nil
		}
		expr_type := c.checkExpr(n.Value)
		if expr_type == nil {
			return nil
		}

		if !expr_type.IsAssignableTo(c.vs.Type(n.Var.Name)) {
			c.error(n.Value, "Type '%s' is not assignable to '%s'.", expr_type.Name(), c.vs.Type(n.Var.Name).Name())
		}

		return expr_type
	case *p.ExprBinary:
		left := c.checkExpr(n.Left)
		if left == nil {
			return nil
		}
		right := c.checkExpr(n.Right)
		if right == nil {
			return nil
		}

		if !IsOneOf(left, TypeNumber{}, TypeString{}, TypeBool{}) || !IsEqual(right, left) {
			c.error(n.Right, "Type '%s' and '%s' aren't compatible.", left.Name(), right.Name())
		}

		return left
	case *p.ExprUnary:
		value := c.checkExpr(n.Value)
		if value == nil {
			return nil
		}

		if !IsOneOf(value, TypeNumber{}, TypeBool{}) {
			c.error(n.Value, "Expected number or bool, got '%s'.", value.Name())
		}

		return value
	case *p.ExprLiteral:
		return n.Value.Type()

	case *p.ExprArrayAccess:
		array_type := c.checkExpr(n.Array)
		if array_type == nil {
			return nil
		}
		for Is[*TypeDefined](array_type) {
			array_type = As[*TypeDefined](array_type).Underlying
		}
		index_type := c.checkExpr(n.Index)
		if index_type == nil {
			return nil
		}
		for Is[*TypeDefined](index_type) {
			index_type = As[*TypeDefined](index_type).Underlying
		}

		if !Is[TypeNumber](index_type) {
			c.errorEnd(n.Index, "Index must be number. Got '%s'.", index_type.Name())
		}
		if !Is[*TypeArray](array_type) {
			c.error(n.Array, "Expected indexed type, got '%s'.", array_type.Name())
		}

		return As[*TypeArray](array_type).ElemType

	default:
		panic(fmt.Sprintf("UNHANDLED CASE ON checkExpr(%s)", code))
	}
}

func (c *Checker) checkTypeExpr(error_node p.Node, type_expr p.ExprTypeRef) Type {
	typ := c.ts.Get(type_expr.Name).Copy()
	if typ == nil {
		c.error(error_node, "Type '%s' not found.", type_expr.Name)
		return nil
	}

	if Is[*TypeArray](typ) {
		t := As[*TypeArray](typ)
		if len(type_expr.Fields) > 0 && t.ElemType != nil {
			c.error(error_node, "Array type doesn't need fields (e.g. %s[%s]) since it's already a declared type.",
				t.Name(), type_expr.Fields[0].Name)
			return nil
		} else if t.ElemType != nil {
			return t
		}

		if len(type_expr.Fields) > 1 {
			c.error(error_node, "Array type only has one single field (e.g. %s[%s]), multiple were provided (e.g. %s[%s, %s...]).",
				t.Name(), type_expr.Fields[0].Name, t.Name(), type_expr.Fields[0].Name, type_expr.Fields[1].Name)
			return nil
		}

		t.ElemType = c.checkTypeExpr(error_node, *type_expr.Fields[0])
		if t.ElemType == nil {
			return nil
		}
		return t
	}

	if Is[*TypeStruct](typ) {
		t := As[*TypeStruct](typ)
		if len(type_expr.Fields) > 0 {
			c.error(error_node, "Struct types doesn't need fields when referenced.")
			return nil
		}

		return t //TODO: implement Struct{fields...} stuff instead of only referencing
	}

	return typ
}
