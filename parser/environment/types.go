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
	TKFunction // TODO: Implement
	TKDefined
)

var (
	NumberType = TypeNumber{}
	BoolType   = TypeBool{}
	StringType = TypeString{}
	ArrayType  = &TypeArray{}
)

type Type interface {
	Name() string
	String() string // for printing
	Copy() Type     // for deep copying
	Kind() TypeKind
	HasFields() bool
	SetFields(fields []Type) error
	AmountFields() int
	HasReturns() bool
	SetReturns(returns []Type) error
	AmountReturns() int
	IsAssignableTo(other Type) bool
}

type TypeNumber struct{}

func (n TypeNumber) Kind() TypeKind                  { return TKNumber }
func (n TypeNumber) Name() string                    { return "Num" }
func (n TypeNumber) String() string                  { return "Num" }
func (n TypeNumber) Copy() Type                      { return NumberType }
func (n TypeNumber) HasFields() bool                 { return false }
func (n TypeNumber) SetFields(fields []Type) error   { return fmt.Errorf("No fields here") }
func (n TypeNumber) AmountFields() int               { return 0 }
func (n TypeNumber) HasReturns() bool                { return false }
func (n TypeNumber) SetReturns(returns []Type) error { return fmt.Errorf("No returns here") }
func (n TypeNumber) AmountReturns() int              { return 0 }
func (n TypeNumber) IsAssignableTo(other Type) bool {
	if other.Kind() == TKDefined {
		return n.IsAssignableTo(other.(*TypeDefined).Underlying)
	}
	return other.Kind() == TKNumber
}

type TypeBool struct{}

func (n TypeBool) Kind() TypeKind                  { return TKBool }
func (n TypeBool) Name() string                    { return "Bool" }
func (n TypeBool) String() string                  { return "Bool" }
func (n TypeBool) Copy() Type                      { return BoolType }
func (n TypeBool) HasFields() bool                 { return false }
func (n TypeBool) SetFields(fields []Type) error   { return fmt.Errorf("No fields here") }
func (n TypeBool) AmountFields() int               { return 0 }
func (n TypeBool) HasReturns() bool                { return false }
func (n TypeBool) SetReturns(returns []Type) error { return fmt.Errorf("No returns here") }
func (n TypeBool) AmountReturns() int              { return 0 }
func (n TypeBool) IsAssignableTo(other Type) bool {
	if other.Kind() == TKDefined {
		return n.IsAssignableTo(other.(*TypeDefined).Underlying)
	}
	return other.Kind() == TKBool
}

type TypeString struct{}

func (n TypeString) Kind() TypeKind                  { return TKString }
func (n TypeString) String() string                  { return "Str" }
func (n TypeString) Copy() Type                      { return StringType }
func (n TypeString) Name() string                    { return "Str" }
func (n TypeString) HasFields() bool                 { return false }
func (n TypeString) SetFields(fields []Type) error   { return fmt.Errorf("No fields here") }
func (n TypeString) AmountFields() int               { return 0 }
func (n TypeString) HasReturns() bool                { return false }
func (n TypeString) SetReturns(returns []Type) error { return fmt.Errorf("No returns here") }
func (n TypeString) AmountReturns() int              { return 0 }
func (n TypeString) IsAssignableTo(other Type) bool {
	if other.Kind() == TKDefined {
		return n.IsAssignableTo(other.(*TypeDefined).Underlying)
	}
	return other.Kind() == TKString
}

type TypeStruct struct {
	TypeName string
	Fields   map[string]Type
}

func (s *TypeStruct) Kind() TypeKind { return TKStruct }
func (s *TypeStruct) Name() string   { return s.TypeName }
func (s *TypeStruct) String() string {
	sb := strings.Builder{}
	sb.WriteString("Struct ")
	sb.WriteString(s.TypeName)
	sb.WriteString(" {\n")
	for name, field := range s.Fields {
		sb.WriteString("  ")
		sb.WriteString(name)
		sb.WriteString(": ")
		sb.WriteString(field.String())
		sb.WriteString(";\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}
func (s *TypeStruct) Copy() Type {
	fields := make(map[string]Type, len(s.Fields))
	for name, field := range s.Fields {
		fields[name] = field.Copy()
	}
	return &TypeStruct{TypeName: s.TypeName, Fields: fields}
}
func (s *TypeStruct) HasFields() bool { return true }
func (s *TypeStruct) SetFields(fields []Type) error {
	for _, field := range fields {
		s.Fields[field.Name()] = field
	}
	return nil
}
func (s *TypeStruct) AmountFields() int               { return len(s.Fields) }
func (s *TypeStruct) HasReturns() bool                { return false }
func (s *TypeStruct) SetReturns(returns []Type) error { return fmt.Errorf("No returns here") }
func (s *TypeStruct) AmountReturns() int              { return 0 }
func (s *TypeStruct) IsAssignableTo(other Type) bool {
	if other.Kind() == TKDefined {
		return s.IsAssignableTo(other.(*TypeDefined).Underlying)
	}
	if other.Kind() != TKStruct {
		return false
	}
	if s.Name() == other.Name() {
		return true
	}

	t := other.(*TypeStruct)
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

func (t *TypeArray) Kind() TypeKind { return TKArray }
func (t *TypeArray) Name() string   { return "Array" }
func (t *TypeArray) String() string {
	return "Array[" + t.ElemType.String() + "]"
}
func (t *TypeArray) Copy() Type {
	if t.ElemType == nil {
		return &TypeArray{ElemType: nil}
	}
	return &TypeArray{ElemType: t.ElemType.Copy()}
}
func (t *TypeArray) HasFields() bool { return true }
func (t *TypeArray) SetFields(fields []Type) error {
	if len(fields) != 1 {
		return fmt.Errorf("Array type must have exactly one field (the element type).")
	}
	t.ElemType = fields[0]
	return nil
}

func (t *TypeArray) AmountFields() int               { return 1 }
func (t *TypeArray) HasReturns() bool                { return false }
func (t *TypeArray) SetReturns(returns []Type) error { return fmt.Errorf("No returns here") }
func (t *TypeArray) AmountReturns() int              { return 0 }
func (t *TypeArray) IsAssignableTo(other Type) bool {
	if other.Kind() == TKDefined {
		return t.IsAssignableTo(other.(*TypeDefined).Underlying)
	}
	if other.Kind() != TKArray {
		return false
	}
	if !t.ElemType.IsAssignableTo(other.(*TypeArray).ElemType) {
		return false
	}
	return true
}

type TypeDefined struct {
	TypeName   string
	Underlying Type
}

func (d *TypeDefined) Kind() TypeKind { return TKDefined }
func (d *TypeDefined) Name() string   { return d.TypeName }
func (d *TypeDefined) String() string { return d.TypeName + ":" + d.Underlying.String() }
func (d *TypeDefined) Copy() Type {
	if d.Underlying == nil {
		return &TypeDefined{TypeName: d.TypeName, Underlying: nil}
	}
	return &TypeDefined{TypeName: d.TypeName, Underlying: d.Underlying.Copy()}
}
func (d *TypeDefined) HasFields() bool                 { return false }
func (d *TypeDefined) SetFields(fields []Type) error   { return fmt.Errorf("No fields here") }
func (d *TypeDefined) AmountFields() int               { return 0 }
func (d *TypeDefined) HasReturns() bool                { return false }
func (d *TypeDefined) SetReturns(returns []Type) error { return fmt.Errorf("No returns here") }
func (d *TypeDefined) AmountReturns() int              { return 0 }
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

func (v *ValueArray) Type() Type { return ArrayType }
func (v *ValueArray) String() string {
	fields := make([]string, 0, len(v.Values))
	for _, val := range v.Values {
		fields = append(fields, val.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(fields, ", "))
}

type ValueDefined struct {
	DefinedType *TypeDefined
	Inner       Value
}

func (v *ValueDefined) Type() Type     { return v.DefinedType }
func (v *ValueDefined) String() string { return v.Inner.String() }
