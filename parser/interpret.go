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

	case *StmtWhile:
		cond, err := Evaluate(stmt.Condition, env)
		if err != nil {
			return 0, err
		}
		if !Is[*ValueBool](cond) {
			return 0, fmt.Errorf("Condition must be a boolean")
		}
		value_bool := *As[*ValueBool](cond)
		for value_bool {
			scope := NewValueScope(env)
			tscope := NewTypeScope(ts)
			_, err = Interpret(stmt.Body, scope, tscope)
			if err != nil {
				return 0, err
			}
			cond, err = Evaluate(stmt.Condition, scope)
			if err != nil {
				return 0, err
			}
			value_bool = *As[*ValueBool](cond)
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

		if e.Op.OneOf(OpNot) && value.Type().IsAssignableTo(BoolType) {
			return new(!*As[*ValueBool](value)), nil
		} else if e.Op.OneOf(OpNot) {
			return nil, fmt.Errorf("Unary operation is not supported on type '%s'", value.Type().Name())
		}

		if !value.Type().IsAssignableTo(NumberType) {
			return nil, fmt.Errorf("Unary operation is not supported on type '%s'", value.Type().Name())
		}
		value_num := *As[*ValueNumber](value)
		switch e.Op.Kind {
		case OpAdd:
			return new(+value_num), nil
		case OpSub:
			return new(-value_num), nil
		case OpBitNot:
			return new(ValueNumber(^int64(value_num))), nil

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

		if e.Op.OneOf(OpAdd) && left.Type().IsAssignableTo(StringType) && right.Type().IsAssignableTo(StringType) {
			result := (*As[*ValueString](left)) + (*As[*ValueString](right))
			return &result, nil
		} else if e.Op.OneOf(OpAdd) && (left.Type().IsAssignableTo(StringType) || right.Type().IsAssignableTo(StringType)) {
			return nil, fmt.Errorf("Concatenation of '%s' and '%s' is not supported", left.Type().Name(), right.Type().Name())
		}

		if e.Op.OneOf(OpAdd, OpSub, OpMul, OpDiv, OpMod, OpBitAnd, OpBitOr, OpBitXor) {
			if !right.Type().IsAssignableTo(NumberType) {
				return nil, fmt.Errorf("Binary operation is not supported on type '%s'", right.Type().Name())
			}
			if !left.Type().IsAssignableTo(NumberType) {
				return nil, fmt.Errorf("Binary operation is not supported on type '%s'", left.Type().Name())
			}

			left_num := *As[*ValueNumber](left)
			right_num := *As[*ValueNumber](right)

			switch e.Op.Kind {
			case OpAdd:
				return new(left_num + right_num), nil
			case OpSub:
				return new(left_num - right_num), nil
			case OpMul:
				return new(left_num * right_num), nil
			case OpDiv:
				return new(left_num / right_num), nil
			case OpMod:
				return new(ValueNumber(math.Mod(float64(left_num), float64(right_num)))), nil
			case OpPow:
				return new(ValueNumber(math.Pow(float64(left_num), float64(right_num)))), nil
			case OpBitAnd:
				return new(ValueNumber(int64(left_num) & int64(right_num))), nil
			case OpBitOr:
				return new(ValueNumber(int64(left_num) | int64(right_num))), nil
			case OpBitXor:
				return new(ValueNumber(int64(left_num) ^ int64(right_num))), nil
			default:
				panic("Invalid operator: " + e.Op.String())
			}
		}

		if e.Op.OneOf(OpGt, OpGe, OpLt, OpLe) {
			if !left.Type().IsAssignableTo(NumberType) {
				return nil, fmt.Errorf("Binary operation is not supported on type '%s'", left.Type().Name())
			}
			if !right.Type().IsAssignableTo(NumberType) {
				return nil, fmt.Errorf("Binary operation is not supported on type '%s'", right.Type().Name())
			}
			left_num := *As[*ValueNumber](left)
			right_num := *As[*ValueNumber](right)

			switch e.Op.Kind {
			case OpGt:
				return new(ValueBool(left_num > right_num)), nil
			case OpGe:
				return new(ValueBool(left_num >= right_num)), nil
			case OpLt:
				return new(ValueBool(left_num < right_num)), nil
			case OpLe:
				return new(ValueBool(left_num <= right_num)), nil
			default:
				panic("Invalid operator: " + e.Op.String())
			}
		}

		if e.Op.OneOf(OpAnd, OpOr) {
			if !left.Type().IsAssignableTo(BoolType) {
				return nil, fmt.Errorf("Left operand of '%s' must be a boolean.", e.Op.String())
			}
			if !right.Type().IsAssignableTo(BoolType) {
				return nil, fmt.Errorf("Right operand of '%s' must be a boolean.", e.Op.String())
			}
			left_bool := *As[*ValueBool](left)
			right_bool := *As[*ValueBool](right)
			switch e.Op.Kind {
			case OpAnd:
				return new(ValueBool(left_bool && right_bool)), nil
			case OpOr:
				return new(ValueBool(left_bool || right_bool)), nil
			default:
				panic("Invalid operator: " + e.Op.String())
			}
		}

		if e.Op.OneOf(OpEq, OpNe) {
			if !(left.Type().IsAssignableTo(NumberType) || left.Type().IsAssignableTo(StringType) || left.Type().IsAssignableTo(BoolType)) {
				return nil, fmt.Errorf("Operands of '%s' must be a number, string or boolean.", e.Op.String())
			}
			if !left.Type().IsAssignableTo(right.Type()) {
				return nil, fmt.Errorf("Operands of '%s' must be the same type.", e.Op.String())
			}

			if left.Type().IsAssignableTo(NumberType) {
				left_num := *As[*ValueNumber](left)
				right_num := *As[*ValueNumber](right)
				switch e.Op.Kind {
				case OpEq:
					return new(ValueBool(left_num == right_num)), nil
				case OpNe:
					return new(ValueBool(left_num != right_num)), nil
				default:
					panic("Invalid operator: " + e.Op.String())
				}
			}
			if left.Type().IsAssignableTo(StringType) {
				left_str := *As[*ValueString](left)
				right_str := *As[*ValueString](right)
				switch e.Op.Kind {
				case OpEq:
					return new(ValueBool(left_str == right_str)), nil
				case OpNe:
					return new(ValueBool(left_str != right_str)), nil
				default:
					panic("Invalid operator: " + e.Op.String())
				}
			}
			if left.Type().IsAssignableTo(BoolType) {
				left_bool := *As[*ValueBool](left)
				right_bool := *As[*ValueBool](right)
				switch e.Op.Kind {
				case OpEq:
					return new(ValueBool(left_bool == right_bool)), nil
				case OpNe:
					return new(ValueBool(left_bool != right_bool)), nil
				default:
					panic("Invalid operator: " + e.Op.String())
				}
			}
		}

		panic("Invalid operator: " + e.Op.String())

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
