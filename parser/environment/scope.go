package environment

type ValueScope struct {
	parent    *ValueScope
	variables map[string]Value
	constants map[string]bool // Which ones of the variables are constants
	existents map[string]bool
}

func NewValueScope(parent *ValueScope) *ValueScope {
	env := &ValueScope{}
	env.parent = parent
	env.variables = make(map[string]Value)
	env.existents = make(map[string]bool)
	env.constants = make(map[string]bool)
	return env
}

func (scp *ValueScope) New(name string, value Value, is_const bool) {
	scp.variables[name] = value
	scp.existents[name] = true
	scp.constants[name] = is_const
}

func (scp *ValueScope) NewConstant(name string, value Value) {
	scp.New(name, value, true)
}

func (scp *ValueScope) NewVar(name string, value Value) {
	scp.New(name, value, false)
}

func (scp *ValueScope) Mark(variable string, is_const bool) {
	scp.existents[variable] = true
	scp.constants[variable] = is_const
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
