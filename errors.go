package datadog

import "fmt"

// Sentinal errors
const (
	ErrMsgPackOverflow Error = iota + 1
)

// Error is a sentinal error
type Error uint64

func (e Error) Error() string {
	switch e {
	case ErrMsgPackOverflow:
		return fmt.Sprintf("maximum msgpack array length (%d) exceeded", MsgPackMaxLength)
	default:
		return "unkown error"
	}
}
