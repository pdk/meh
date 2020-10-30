package compile

// Tuple is distinct from a slice.
type Tuple struct {
	Values []interface{}
}

// NewTuple returns a new Tuple.
func NewTuple(values ...interface{}) Tuple {
	return Tuple{
		Values: values,
	}
}
