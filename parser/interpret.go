package parser

import (
	. "smol/lexer/token"
	. "smol/parser/environment"

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
			if _, is_return := s.(*StmtReturn); is_return {
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
			if _, is_return := s.(*StmtReturn); is_return {
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
		value_float := value.(ValueNumber)
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
		value_bool := value.(ValueBool)

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
		if !array.Type().Kind().OneOf(TKArray) {
			return nil, fmt.Errorf("Expected an array.")
		}
		index, err := Evaluate(e.Index, env)
		if err != nil {
			return nil, err
		}
		if !index.Type().Kind().OneOf(TKNumber) {
			return nil, fmt.Errorf("Expected a number index.")
		}
		values := array.(*ValueArray).Values
		i := index.(ValueNumber)
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
		value_num := value.(ValueNumber)
		switch e.Op.Kind {
		case OpAdd:
			return +value_num, nil
		case OpSub:
			return -value_num, nil

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

		if right.Type().Kind() == TKString && left.Type().Kind() == TKString {
			return left.(ValueString) + right.(ValueString), nil
		}
		if right.Type().Kind() == TKString || left.Type().Kind() == TKString {
			return nil, fmt.Errorf("Concatenation of '%s' and '%s' is not supported", left.Type().Name(), right.Type().Name())
		}

		if !right.Type().IsAssignableTo(NumberType) {
			return nil, fmt.Errorf("Binary operation is not supported on type '%s'", right.Type().Name())
		}
		if !left.Type().IsAssignableTo(NumberType) {
			return nil, fmt.Errorf("Binary operation is not supported on type '%s'", left.Type().Name())
		}

		left_num := left.(ValueNumber)
		right_num := right.(ValueNumber)

		switch e.Op.Kind {
		case OpAdd:
			return left_num + right_num, nil
		case OpSub:
			return left_num - right_num, nil
		case OpMul:
			return left_num * right_num, nil
		case OpDiv:
			return left_num / right_num, nil
		case OpMod:
			return ValueNumber(math.Mod(float64(left_num), float64(right_num))), nil
		case OpPow:
			return ValueNumber(math.Pow(float64(left_num), float64(right_num))), nil

		default:
			return nil, fmt.Errorf("Invalid operator: %s", e.Op)
		}
	default:
		panic("Unknown node: " + expr.String())
	}
}

// same function as in checkers
func checkTypeExpr(type_expr ExprTypeRef, ts *TypeScope) (Type, error) {
	typ := ts.Get(type_expr.Name)
	if typ == nil {
		return nil, fmt.Errorf("Type '%s' not found.", type_expr.Name)
	}

	if !typ.HasFields() && len(type_expr.Fields) > 0 {
		return nil, fmt.Errorf("Type '%s' does not have fields.", typ.Name())
	}

	if !typ.HasReturns() && len(type_expr.Returns) > 0 {
		return nil, fmt.Errorf("Type '%s' does not have returns.", typ.Name())
	}

	if len(type_expr.Fields) > 0 {
		fields := make([]Type, 0, len(type_expr.Fields))
		for _, field := range type_expr.Fields {
			field_typ, err := checkTypeExpr(*field, ts)
			if err != nil {
				return nil, err
			}
			fields = append(fields, field_typ)
		}
		err := typ.SetFields(fields)
		if err != nil {
			return nil, fmt.Errorf("Failed to set fields for type '%s': %v.", typ.Name(), err)
		}
	}

	if len(type_expr.Returns) > 0 {
		returns := make([]Type, 0, len(type_expr.Returns))
		for _, return_expr := range type_expr.Returns {
			return_typ, err := checkTypeExpr(*return_expr, ts)
			if err != nil {
				return nil, err
			}
			returns = append(returns, return_typ)
		}
		err := typ.SetReturns(returns)
		if err != nil {
			return nil, fmt.Errorf("Failed to set returns for type '%s': %v.", typ.Name(), err)
		}
	}

	return typ, nil
}
