package parser

import (
	. "smol/lexer/token"
	. "smol/parser/environment"
	. "smol/util"

	"fmt"
	"math"
)

func Interpret(code Statement, env *ValueScope, ts *TypeScope) (float64, error) {
	if env == nil {
		return 0, fmt.Errorf("Environment is nil.")
	}
	if code == nil {
		panic("Code is nil.")
	}

	switch stmt := code.(type) {
	case *StmtEmpty:
		break

	case *Program:
		for _, s := range stmt.Children {
			exit_code, err := Interpret(s, env, ts)
			if err != nil {
				return 0, err
			}
			if Is[*StmtReturn](s) {
				return exit_code, nil
			}
		}
		return 0, nil

	case *StmtBlock:
		scope := NewValueScope(env)
		tscope := NewTypeScope(ts)
		for _, s := range stmt.Children {
			exit_code, err := Interpret(s, scope, tscope)
			if err != nil {
				return 0, err
			}
			if Is[*StmtReturn](s) {
				return exit_code, nil
			}
		}
		return 0, nil

	case *StmtExpression:
		_, err := Evaluate(stmt.Expr, env)
		if err != nil {
			return 0, err
		}

	case *StmtPrint:
		values := make([]Value, 0)
		for _, expr := range stmt.Values {
			value, err := Evaluate(expr, env)
			if err != nil {
				return 0, err
			}
			values = append(values, value)
		}

		for _, value := range values {
			fmt.Print(value)
			fmt.Print(" ")
		}
		if stmt.IsPrintln {
			fmt.Println()
		}

	case *StmtReturn:
		value, err := Evaluate(stmt.Expr, env)
		if err != nil {
			return 0, err
		}
		if !value.Type().IsAssignableTo(NumberType) {
			return 0, fmt.Errorf("Return value must be a number")
		}
		value_float := *As[*ValueNumber](value)
		return float64(value_float), nil

	case *StmtAssign:
		var value Value = nil
		var typ Type = nil
		var err error = nil
		if stmt.Value != nil {
			value, err = Evaluate(stmt.Value, env)
		} else {
			typ, err = checkTypeExpr(*stmt.TypeExpr, ts)
		}
		if err != nil {
			return 0, err
		}
		env.New(stmt.Var.Name, value, stmt.Var.IsConst, typ)

	case *StmtIf:
		value, err := Evaluate(stmt.Condition, env)
		if err != nil {
			return 0, err
		}
		if !value.Type().IsAssignableTo(BoolType) {
			return 0, fmt.Errorf("Condition must be a boolean")
		}
		value_bool := *As[*ValueBool](value)

		if value_bool {
			scope := NewValueScope(env)
			tscope := NewTypeScope(ts)
			_, err = Interpret(stmt.If, scope, tscope)
			if err != nil {
				return 0, err
			}
		} else if stmt.Else != nil {
			scope := NewValueScope(env)
			tscope := NewTypeScope(ts)
			_, err = Interpret(stmt.Else, scope, tscope)
			if err != nil {
				return 0, err
			}
		}

	case *StmtDeclType:
		typ, err := checkTypeExpr(*stmt.TypeExpr, ts)
		if err != nil {
			return 0, err
		}
		ts.Set(stmt.Name, typ)

	default:
		panic("Unknown node: " + stmt.String())
	}

	return 0, nil
}

func Evaluate(expr Expression, env *ValueScope) (Value, error) {
	if env == nil {
		return nil, fmt.Errorf("Environment is nil.")
	}
	if expr == nil {
		panic("Expression is nil.")
	}

	switch e := expr.(type) {
	case *ExprLiteral:
		return e.Value, nil

	case *ExprGroup: // dis but a mere wrapper
		return Evaluate(e.Value, env)
	case *ExprArray:
		values := make([]Value, len(e.Fields))
		for i, field := range e.Fields {
			value, err := Evaluate(field, env)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return &ValueArray{Values: values}, nil
	case *ExprArrayAccess:
		array, err := Evaluate(e.Array, env)
		if err != nil {
			return nil, err
		}
		if !Is[*TypeArray](array.Type()) {
			return nil, fmt.Errorf("Expected an array.")
		}
		index, err := Evaluate(e.Index, env)
		if err != nil {
			return nil, err
		}
		if !Is[*TypeNumber](index.Type()) {
			return nil, fmt.Errorf("Expected a number index.")
		}
		values := As[*ValueArray](array).Values
		i := *As[*ValueNumber](index)
		if float64(i) >= float64(len(values)) {
			return nil, fmt.Errorf("Index out of bounds.")
		}
		if float64(i) < 0 {
			return nil, fmt.Errorf("Negative index.")
		}
		return values[int(i)], nil
	case *ExprAssign:
		if e.Var.IsConst {
			return nil, fmt.Errorf("Cannot assign a new value to a constant.")
		}
		if !env.Exists(e.Var.Name) {
			return nil, fmt.Errorf("Cannot assign to non-existent variable.")
		}

		value, err := Evaluate(e.Value, env)
		if err != nil {
			return nil, err
		}
		var_type := env.Get(e.Var.Name).Type()

		if !value.Type().IsAssignableTo(var_type) {
			return nil, fmt.Errorf("Cannot assign value of type '%s' to variable of type '%s'", value.Type().Name(), var_type.Name())
		}

		env.Set(e.Var.Name, value)
		return value, nil

	case *ExprVariable:
		value := env.Get(e.Name)
		return value, nil

	case *ExprUnary:
		value, err := Evaluate(e.Value, env)
		if err != nil {
			return nil, err
		}

		if !value.Type().IsAssignableTo(NumberType) {
			return nil, fmt.Errorf("Unary operation is not supported on type '%s'", value.Type().Name())
		}
		value_num := *As[*ValueNumber](value)
		switch e.Op.Kind {
		case OpAdd:
			result := +value_num
			return &result, nil
		case OpSub:
			result := -value_num
			return &result, nil

		default:
			return nil, fmt.Errorf("Invalid unary operator: %s", e.Op)
		}

	case *ExprBinary:
		left, err := Evaluate(e.Left, env)
		if err != nil {
			return nil, err
		}

		right, err := Evaluate(e.Right, env)
		if err != nil {
			return nil, err
		}

		if Is[*TypeString](right.Type()) && Is[*TypeString](left.Type()) {
			result := (*As[*ValueString](left)) + (*As[*ValueString](right))
			return &result, nil
		}
		if Is[*TypeString](right.Type()) || Is[*TypeString](left.Type()) {
			return nil, fmt.Errorf("Concatenation of '%s' and '%s' is not supported", left.Type().Name(), right.Type().Name())
		}

		if !right.Type().IsAssignableTo(NumberType) {
			return nil, fmt.Errorf("Binary operation is not supported on type '%s'", right.Type().Name())
		}
		if !left.Type().IsAssignableTo(NumberType) {
			return nil, fmt.Errorf("Binary operation is not supported on type '%s'", left.Type().Name())
		}

		left_num := As[*ValueNumber](left)
		right_num := As[*ValueNumber](right)

		switch e.Op.Kind {
		case OpAdd:
			result := *left_num + *right_num
			return &result, nil
		case OpSub:
			result := *left_num - *right_num
			return &result, nil
		case OpMul:
			result := *left_num * *right_num
			return &result, nil
		case OpDiv:
			result := *left_num / *right_num
			return &result, nil
		case OpMod:
			result := ValueNumber(math.Mod(float64(*left_num), float64(*right_num)))
			return &result, nil
		case OpPow:
			result := ValueNumber(math.Pow(float64(*left_num), float64(*right_num)))
			return &result, nil

		default:
			return nil, fmt.Errorf("Invalid operator: %s", e.Op)
		}
	default:
		panic("Unknown node: " + expr.String())
	}
}

// same function as in checkers
func checkTypeExpr(type_expr ExprTypeRef, ts *TypeScope) (Type, error) {
	var err error
	typ := ts.Get(type_expr.Name).Copy()
	if typ == nil {
		return nil, fmt.Errorf("Type '%s' not found.", type_expr.Name)
	}

	if Is[*TypeArray](typ) {
		t := As[*TypeArray](typ)
		if len(type_expr.Fields) > 0 && t.ElemType != nil {
			return nil, fmt.Errorf("Array type '%s' doesn't need fields since it's already a declared type.", t.Name())
		} else if t.ElemType != nil {
			return t, nil
		}

		if len(type_expr.Fields) > 1 {
			return nil, fmt.Errorf("Array type can only have one single field (its elem type), multiple were provided.")
		}

		t.ElemType, err = checkTypeExpr(*type_expr.Fields[0], ts)
		if err != nil {
			return nil, err
		}
		return t, nil
	}

	if Is[*TypeStruct](typ) {
		t := As[*TypeStruct](typ)
		if len(type_expr.Fields) > 0 {
			return nil, fmt.Errorf("Struct types don't need fields when referenced.")
		}

		return t, nil //TODO: implement Struct{fields...} stuff instead of only referencing
	}

	return typ, nil
}
