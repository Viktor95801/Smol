package environment

import (
	"fmt"
	"slices"
	"strings"
)

type TypeKind int

func (k TypeKind) OneOf(kinds ...TypeKind) bool {
	return slices.Contains(kinds, k)
}

const (
	TKNumber TypeKind = iota
	TKBool
	TKString
	TKArray
	TKStruct
	TKDefined
)

var (
	NumberType = TypeNumber{}
	BoolType   = TypeBool{}
	StringType = TypeString{}
	TupleType  = TypeArray{}
)

type Type interface {
	Name() string
	Kind() TypeKind
	HasFields() bool
	SetFields(fields []Type)
	HasReturns() bool
	SetReturns(returns []Type)
	IsAssignableTo(other Type) bool
}

type TypeNumber struct{}

func (n TypeNumber) Kind() TypeKind            { return TKNumber }
func (n TypeNumber) Name() string              { return "Num" }
func (n TypeNumber) HasFields() bool           { return false }
func (n TypeNumber) SetFields(fields []Type)   { panic("No fields here") }
func (n TypeNumber) HasReturns() bool          { return false }
func (n TypeNumber) SetReturns(returns []Type) { panic("No returns here") }
func (n TypeNumber) IsAssignableTo(other Type) bool {
	return other.Kind() == TKNumber
}

type TypeBool struct{}

func (n TypeBool) Kind() TypeKind            { return TKBool }
func (n TypeBool) Name() string              { return "Bool" }
func (n TypeBool) HasFields() bool           { return false }
func (n TypeBool) SetFields(fields []Type)   { panic("No fields here") }
func (n TypeBool) HasReturns() bool          { return false }
func (n TypeBool) SetReturns(returns []Type) { panic("No returns here") }
func (n TypeBool) IsAssignableTo(other Type) bool {
	return other.Kind() == TKBool
}

type TypeString struct{}

func (n TypeString) Kind() TypeKind            { return TKString }
func (n TypeString) Name() string              { return "Str" }
func (n TypeString) HasFields() bool           { return false }
func (n TypeString) SetFields(fields []Type)   { panic("No fields here") }
func (n TypeString) HasReturns() bool          { return false }
func (n TypeString) SetReturns(returns []Type) { panic("No returns here") }
func (n TypeString) IsAssignableTo(other Type) bool {
	return other.Kind() == TKString
}

type TypeStruct struct {
	TypeName string
	Fields   map[string]Type
}

func (s *TypeStruct) Kind() TypeKind  { return TKStruct }
func (s *TypeStruct) Name() string    { return s.TypeName }
func (s *TypeStruct) HasFields() bool { return true }
func (s *TypeStruct) SetFields(fields []Type) {
	for _, field := range fields {
		s.Fields[field.Name()] = field
	}
}
func (s *TypeStruct) HasReturns() bool          { return false }
func (s *TypeStruct) SetReturns(returns []Type) { panic("No returns here") }
func (s *TypeStruct) IsAssignableTo(target Type) bool {
	if target.Kind() != TKStruct {
		return false
	}
	if s.Name() == target.Name() {
		return true
	}

	t := target.(*TypeStruct)
	for field, typ := range s.Fields {
		if !typ.IsAssignableTo(t.Fields[field]) {
			return false
		}
	}
	return true
}

type TypeArray struct {
	ElemType Type
}

func (t TypeArray) Kind() TypeKind            { return TKArray }
func (t TypeArray) Name() string              { return "Array" }
func (t TypeArray) HasFields() bool           { return false }
func (t TypeArray) SetFields(fields []Type)   { panic("No fields here") }
func (t TypeArray) HasReturns() bool          { return false }
func (t TypeArray) SetReturns(returns []Type) { panic("No returns here") }
func (t TypeArray) IsAssignableTo(target Type) bool {
	if target.Kind() != TKArray {
		return false
	}
	if !t.ElemType.IsAssignableTo(target.(TypeArray).ElemType) {
		return false
	}
	return true
}

type TypeDefined struct {
	TypeName   string
	Underlying Type
}

func (d *TypeDefined) Kind() TypeKind            { return TKDefined }
func (d *TypeDefined) Name() string              { return d.TypeName }
func (d *TypeDefined) HasFields() bool           { return false }
func (d *TypeDefined) SetFields(fields []Type)   { panic("No fields here") }
func (d *TypeDefined) HasReturns() bool          { return false }
func (d *TypeDefined) SetReturns(returns []Type) { panic("No returns here") }
func (d *TypeDefined) IsAssignableTo(target Type) bool {
	if d.Name() == target.Name() {
		return true
	}
	return d.Underlying.IsAssignableTo(target) // implicit casting
	// return false // explicit casting
}

type Value interface {
	Type() Type
	String() string
}

type ValueNumber float64

func (v ValueNumber) Type() Type     { return NumberType }
func (v ValueNumber) String() string { return fmt.Sprintf("%g", v) }

type ValueBool bool

func (v ValueBool) Type() Type     { return BoolType }
func (v ValueBool) String() string { return fmt.Sprintf("%t", v) }

type ValueString string

func (v ValueString) Type() Type     { return StringType }
func (v ValueString) String() string { return string(v) }

type ValueStruct struct {
	StructType *TypeStruct
	Fields     map[string]Value
}

func (v *ValueStruct) Type() Type { return v.StructType }
func (v *ValueStruct) String() string {
	var fields []string
	for k, val := range v.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", k, val.String()))
	}
	return fmt.Sprintf("%s{%s}", v.StructType.Name(), strings.Join(fields, ", "))
}

type ValueArray struct {
	Values []Value
}

func (v *ValueArray) Type() Type { return TupleType }
func (v *ValueArray) String() string {
	fields := make([]string, 0, len(v.Values))
	for _, val := range v.Values {
		fields = append(fields, val.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(fields, ", "))
}

type ValueDefined struct {
	DefinedType *TypeDefined
	Inner       Value
}

func (v *ValueDefined) Type() Type     { return v.DefinedType }
func (v *ValueDefined) String() string { return v.Inner.String() }
