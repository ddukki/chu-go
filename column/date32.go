package column

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

// Date32Column is a Date32 column (stored as days since epoch, int32).
type Date32Column struct {
	name string
	Data []int32
}

// NewDate32Column creates a Date32Column with the given column name.
func NewDate32Column(name string) *Date32Column {
	return &Date32Column{name: name}
}

func (c *Date32Column) Name() string                  { return c.name }
func (c *Date32Column) Type() proto.ColumnType         { return proto.ColumnTypeDate32 }
func (c *Date32Column) Len() int                       { return len(c.Data) }
func (c *Date32Column) Reset()                         { c.Data = c.Data[:0] }

func (c *Date32Column) Append(v time.Time) {
	c.Data = append(c.Data, int32(v.Unix()/86400))
}

func (c *Date32Column) AppendArr(vs []time.Time) {
	for _, v := range vs {
		c.Data = append(c.Data, int32(v.Unix()/86400))
	}
}

func (c *Date32Column) Row(i int) time.Time {
	return time.Unix(int64(c.Data[i])*86400, 0).UTC()
}

func (c *Date32Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 4, &c.Data)
}

func (c *Date32Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 4, b)
}

func (c *Date32Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 4, w)
}
