package environment

import (
	. "smol/util"

	"fmt"
	"strings"
)

var (
	NumberType = &TypeNumber{}
	BoolType   = &TypeBool{}
	StringType = &TypeString{}
	ArrayType  = &TypeArray{}
)

type Type interface {
	Name() string
	String() string // for printing
	Copy() Type     // for deep copying
	IsAssignableTo(other Type) bool
}

type TypeNumber struct{}

func (n *TypeNumber) Name() string   { return "Num" }
func (n *TypeNumber) String() string { return "Num" }
func (n *TypeNumber) Copy() Type     { return NumberType }
func (n *TypeNumber) IsAssignableTo(other Type) bool {
	if Is[*TypeDefined](other) {
		return n.IsAssignableTo(As[*TypeDefined](other).Underlying)
	}
	return Is[*TypeNumber](other)
}

type TypeBool struct{}

func (n *TypeBool) Name() string   { return "Bool" }
func (n *TypeBool) String() string { return "Bool" }
func (n *TypeBool) Copy() Type     { return BoolType }
func (n *TypeBool) IsAssignableTo(other Type) bool {
	if Is[*TypeDefined](other) {
		return n.IsAssignableTo(As[*TypeDefined](other).Underlying)
	}
	return Is[*TypeBool](other)
}

type TypeString struct{}

func (n *TypeString) String() string { return "Str" }
func (n *TypeString) Copy() Type     { return StringType }
func (n *TypeString) Name() string   { return "Str" }
func (n *TypeString) IsAssignableTo(other Type) bool {
	if Is[*TypeDefined](other) {
		return n.IsAssignableTo(As[*TypeDefined](other).Underlying)
	}
	return Is[*TypeString](other)
}

type TypeStruct struct {
	TypeName string
	Fields   map[string]Type
}

func (s *TypeStruct) Name() string { return s.TypeName }
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
func (s *TypeStruct) IsAssignableTo(other Type) bool {
	if Is[*TypeDefined](other) {
		return s.IsAssignableTo(As[*TypeDefined](other).Underlying)
	}
	if !Is[*TypeStruct](other) {
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

func (t *TypeArray) Name() string { return "Array" }
func (t *TypeArray) String() string {
	return "Array[" + t.ElemType.String() + "]"
}
func (t *TypeArray) Copy() Type {
	if t.ElemType == nil {
		return &TypeArray{ElemType: nil}
	}
	return &TypeArray{ElemType: t.ElemType.Copy()}
}
func (t *TypeArray) IsAssignableTo(other Type) bool {
	if Is[*TypeDefined](other) {
		return t.IsAssignableTo(As[*TypeDefined](other).Underlying)
	}
	if !Is[*TypeArray](other) {
		return false
	}
	if !t.ElemType.IsAssignableTo(As[*TypeArray](other).ElemType) {
		return false
	}
	return true
}

type TypeDefined struct {
	TypeName   string
	Underlying Type
}

func (d *TypeDefined) Name() string   { return d.TypeName }
func (d *TypeDefined) String() string { return d.TypeName + ":" + d.Underlying.String() }
func (d *TypeDefined) Copy() Type {
	if d.Underlying == nil {
		return &TypeDefined{TypeName: d.TypeName, Underlying: nil}
	}
	return &TypeDefined{TypeName: d.TypeName, Underlying: d.Underlying.Copy()}
}
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

func (v *ValueNumber) Type() Type     { return NumberType }
func (v *ValueNumber) String() string { return fmt.Sprintf("%g", *v) }

type ValueBool bool

func (v *ValueBool) Type() Type     { return BoolType }
func (v *ValueBool) String() string { return fmt.Sprintf("%t", *v) }

type ValueString string

func (v *ValueString) Type() Type     { return StringType }
func (v *ValueString) String() string { return string(*v) }

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
