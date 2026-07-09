package conn

import "github.com/ClickHouse/ch-go/proto"

type Column interface {
	Name() string
	Type() proto.ColumnType
	Len() int
	DecodeColumn(r *proto.Reader, rows int) error
	EncodeColumn(b *proto.Buffer) error
	WriteColumn(w *proto.Writer)
}
