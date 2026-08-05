package checkers

import (
	p "smol/parser"
	. "smol/parser/environment"

	//	. "smol/lexer/token"

	"fmt"
)

type Checker struct {
	Errors []error

	file     *string
	vs       *ValueScope
	ts_types *TypeScope
	ts_vars  *TypeScope
}

func NewChecker(file string, vs *ValueScope, ts *TypeScope) *Checker {
	c := &Checker{}
	c.Init(&file, vs, ts)
	return c
}

func (c *Checker) Init(file *string, vs *ValueScope, ts *TypeScope) {
	c.file = file
	c.vs = vs
	c.ts_types = ts
	c.ts_vars = NewTypeScope(ts)
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
		parent_ts_types := c.ts_types
		parent_ts_vars := c.ts_vars

		c.vs = NewValueScope(parent_vs)
		c.ts_types = NewTypeScope(parent_ts_types)
		c.ts_vars = NewTypeScope(parent_ts_vars)
		for _, child := range n.Children {
			c.Check(child)
		}
		c.vs = parent_vs
		c.ts_types = parent_ts_types
		c.ts_vars = parent_ts_vars
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
				c.error(n.Var, "Cannot assign '%s' to '%s'.", value.Name(), typ.Name())
				return
			}
			typ = value
		}

		c.vs.Mark(n.Var.Name, n.Var.Const)
		c.ts_vars.Set(n.Var.Name, typ)
	case *p.StmtExpression:
		c.checkExpr(n.Expr)
	case *p.StmtIf:
		cond := c.checkExpr(n.Condition)
		if cond == nil {
			return
		}
		if !cond.Kind().OneOf(TKBool) {
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

		if !expr.Kind().OneOf(TKNumber) {
			c.error(n.Expr, "Expected number, got '%s'.", expr.Name())
		}
	case *p.StmtPrint:
		for _, value := range n.Values {
			c.checkExpr(value)
		}

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
		n.Const = c.vs.IsConst(n.Name)
		return c.ts_vars.Get(n.Name)
	case *p.ExprAssign:
		if !c.vs.Exists(n.Var.Name) {
			c.error(n.Var, "Undefined variable '%s'.", n.Var.Name)
			return nil
		}
		expr_type := c.checkExpr(n.Value)
		if expr_type == nil {
			return nil
		}

		if !expr_type.IsAssignableTo(c.ts_vars.Get(n.Var.Name)) {
			c.error(n.Value, "Type '%s' is not assignable to '%s'.", expr_type.Name(), c.ts_vars.Get(n.Var.Name).Name())
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

		if !left.Kind().OneOf(TKNumber, TKString, TKBool) || !left.Kind().OneOf(right.Kind()) {
			c.error(n.Right, "Type '%s' and '%s' aren't compatible.", left.Name(), right.Name())
		}

		return left
	case *p.ExprUnary:
		value := c.checkExpr(n.Value)
		if value == nil {
			return nil
		}

		if !value.Kind().OneOf(TKNumber, TKBool) {
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
		index_type := c.checkExpr(n.Index)
		if index_type == nil {
			return nil
		}

		if !index_type.Kind().OneOf(TKNumber) {
			c.errorEnd(n.Index, "Index must be number. Got '%s'.", index_type.Name())
		}
		if !array_type.Kind().OneOf(TKArray) {
			c.error(n.Array, "Expected indexed type, got '%s'.", array_type.Name())
		}

		return array_type.(*TypeArray).ElemType

	default:
		panic(fmt.Sprintf("UNHANDLED CASE ON checkExpr(%s)", code))
	}
}

func (c *Checker) checkTypeExpr(error_node p.Node, type_expr p.TypeExpression) Type {
	typ := c.ts_types.Get(type_expr.Name)
	if typ == nil {
		c.error(error_node, "Type '%s' not found.", type_expr.Name)
		return nil
	}

	if !typ.HasFields() && len(type_expr.Fields) > 0 {
		c.error(error_node, "Type '%s' does not have fields.", typ.Name())
		return nil
	}

	if !typ.HasReturns() && len(type_expr.Returns) > 0 {
		c.error(error_node, "Type '%s' does not have returns.", typ.Name())
		return nil
	}

	if len(type_expr.Fields) > 0 {
		fields := make([]Type, 0, len(type_expr.Fields))
		for _, field := range type_expr.Fields {
			field_typ := c.checkTypeExpr(error_node, *field)
			if field_typ == nil {
				return nil
			}
			fields = append(fields, field_typ)
		}
		err := typ.SetFields(fields)
		if err != nil {
			c.error(error_node, "Failed to set fields for type '%s': %v.", typ.Name(), err)
			return nil
		}
	}

	if len(type_expr.Returns) > 0 {
		returns := make([]Type, 0, len(type_expr.Returns))
		for _, return_expr := range type_expr.Returns {
			return_typ := c.checkTypeExpr(error_node, *return_expr)
			if return_typ == nil {
				return nil
			}
			returns = append(returns, return_typ)
		}
		typ.SetReturns(returns)
	}

	return typ
}
