package compile

// FlowChangeType indicates the type of flow control change.
type FlowChangeType byte

// Which kinds of flow control changes that are handled.
const (
	None FlowChangeType = iota
	Return
	Break
	Continue
)

// FlowChange is what is returned by an Expr when there is a change of flow.
type FlowChange struct {
	Type FlowChangeType
	Value
}

// flowChange checks if the value is a FlowChange.
func flowChange(v Value) FlowChangeType {
	change, ok := v.(FlowChange)
	if !ok {
		return None
	}
	return change.Type
}

// NewReturn produces a Return FlowChange.
func NewReturn(values ...Value) Value {

	switch len(values) {
	case 0:
		return FlowChange{
			Type:  Return,
			Value: nil,
		}
	case 1:
		return FlowChange{
			Type:  Return,
			Value: values[0],
		}
	default:
		return FlowChange{
			Type:  Return,
			Value: values,
		}
	}
}

// NewBreak produces a Break FlowChange.
func NewBreak() Value {
	return FlowChange{Type: Break}
}

// NewContinue produces a Continue FlowChange.
func NewContinue() Value {
	return FlowChange{Type: Continue}
}
