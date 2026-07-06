package conn

import "fmt"

type Kind int

const (
	KindNetwork  Kind = iota
	KindProtocol
	KindConfig
	KindServer
	KindInternal
)

type Error struct {
	Kind    Kind
	Message string
	ServerCode int
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }
