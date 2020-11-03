package compile

// Context is the current name->value map.
type Context struct {
	values map[string]Value
	parent *Context
}

// NewTopContext returns a new top context.
func NewTopContext() *Context {
	ctx := NewContext(nil)
	// todo: add global things
	return ctx
}

// NewContext returns a new context.
func NewContext(parent *Context) *Context {
	return &Context{
		values: make(map[string]Value),
		parent: parent,
	}
}

// Set sets a variable to a new value. Might return error, e.g. illegal type
// change.
func (ctx *Context) Set(name string, value Value) (Value, error) {
	ctx.values[name] = value
	return value, nil
}

// Get returns the current value for the variable named, or nil if not assigned.
func (ctx *Context) Get(name string) Value {

	if ctx == nil {
		return nil
	}

	val, ok := ctx.values[name]
	if !ok {
		return ctx.parent.Get(name)
	}

	return val
}
