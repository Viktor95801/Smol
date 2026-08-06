package environment

type ValueScope struct {
	parent    *ValueScope
	variables map[string]Value
	vars_type map[string]Type
	constants map[string]bool // Which ones of the variables are constants
	existents map[string]bool
}

func NewValueScope(parent *ValueScope) *ValueScope {
	env := &ValueScope{}
	env.parent = parent
	env.variables = make(map[string]Value)
	env.vars_type = make(map[string]Type)
	env.existents = make(map[string]bool)
	env.constants = make(map[string]bool)
	return env
}

func (scp *ValueScope) New(name string, value Value, is_const bool, typ ...Type) {
	if len(typ) > 1 {
		panic("Too many types on ValueScope.New()")
	}
	if value != nil {
		scp.variables[name] = value
		scp.vars_type[name] = value.Type()
		if len(typ) > 0 && typ[0] != nil && value.Type() != typ[0] {
			panic("Type mismatch on ValueScope.New() value type and typ")
		}
	} else if len(typ) == 1 {
		scp.variables[name] = defaultValue(typ[0])
		scp.vars_type[name] = typ[0]
	} else {
		panic("Too many types on ValueScope.New()")
	}
	scp.existents[name] = true
	scp.constants[name] = is_const
}

func (scp *ValueScope) NewConstant(name string, value Value) {
	scp.New(name, value, true)
}

func (scp *ValueScope) NewVar(name string, value Value) {
	scp.New(name, value, false)
}

func (scp *ValueScope) Mark(variable string, is_const bool, typ Type) {
	scp.existents[variable] = true
	scp.constants[variable] = is_const
	scp.vars_type[variable] = typ
}

func (scp *ValueScope) Set(variable string, value Value) bool {
	if scp.constants[variable] {
		return false
	}
	if !scp.existents[variable] {
		if scp.parent == nil {
			return false
		}
		return scp.parent.Set(variable, value)
	}

	if !value.Type().IsAssignableTo(scp.vars_type[variable]) {
		panic("Stactic type checking failed")
	}

	scp.variables[variable] = value
	return true
}

func (scp *ValueScope) IsConst(name string) bool {
	if !scp.existents[name] {
		if scp.parent == nil {
			return false
		}
		return scp.parent.IsConst(name)
	}
	return scp.constants[name]
}

func (scp *ValueScope) Exists(variable string) bool {
	if !scp.existents[variable] {
		if scp.parent == nil {
			return false
		}

		return scp.parent.Exists(variable)
	}
	return true
}

func (scp *ValueScope) Type(variable string) Type {
	if !scp.existents[variable] {
		if scp.parent == nil {
			return nil
		}
		return scp.parent.Type(variable)
	}
	return scp.vars_type[variable]
}

func (scp *ValueScope) Get(variable string) Value {
	result, exists := scp.variables[variable]
	if !exists {
		if scp.parent == nil {
			return nil
		}
		return scp.parent.Get(variable)
	}
	return result
}

func defaultValue(typ Type) Value {
	switch t := typ.(type) {
	case *TypeNumber:
		return new(ValueNumber(0))
	case *TypeBool:
		return new(ValueBool(false))
	case *TypeString:
		return new(ValueString(""))
	case *TypeArray:
		return new(ValueArray{Values: []Value{}})
	case *TypeStruct:
		fields := make(map[string]Value)
		for name, fieldTyp := range t.Fields {
			fields[name] = defaultValue(fieldTyp)
		}
		return new(ValueStruct{Fields: fields})
	case *TypeDefined:
		return new(ValueDefined{
			DefinedType: t,
			Inner:       defaultValue(t.Underlying),
		})
	default:
		panic("Unknown type")
	}
}

type TypeScope struct {
	parent *TypeScope
	types  map[string]Type
}

func NewTypeScope(parent *TypeScope) *TypeScope {
	scp := &TypeScope{}
	scp.Init(parent)
	return scp
}

func (scp *TypeScope) Init(parent *TypeScope) {
	scp.parent = parent
	scp.types = make(map[string]Type)
}

func (scp *TypeScope) Set(name string, typ Type) {
	scp.types[name] = typ
}

func (scp *TypeScope) Get(name string) Type {
	typ, exists := scp.types[name]
	if !exists {
		if scp.parent == nil {
			return nil
		}
		return scp.parent.Get(name)
	}
	return typ
}
