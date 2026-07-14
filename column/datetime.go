package column

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

// DateTimeColumn is a DateTime column (stored as UInt32).
type DateTimeColumn struct {
	name string
	Data []uint32
}

// NewDateTimeColumn creates a DateTimeColumn with the given column name.
func NewDateTimeColumn(name string) *DateTimeColumn {
	return &DateTimeColumn{name: name}
}

// Name returns the column name.
func (c *DateTimeColumn) Name() string { return c.name }

// Type returns proto.ColumnTypeDateTime.
func (c *DateTimeColumn) Type() proto.ColumnType { return proto.ColumnTypeDateTime }

// Len returns the number of elements in the column.
func (c *DateTimeColumn) Len() int { return len(c.Data) }

// Append adds a single time value, stored as Unix timestamp.
func (c *DateTimeColumn) Append(v time.Time) {
	c.Data = append(c.Data, uint32(v.Unix()))
}

// AppendArr adds multiple time values.
func (c *DateTimeColumn) AppendArr(vs []time.Time) {
	for _, v := range vs {
		c.Data = append(c.Data, uint32(v.Unix()))
	}
}

// Row returns the time value at index i.
func (c *DateTimeColumn) Row(i int) time.Time {
	return time.Unix(int64(c.Data[i]), 0)
}

// Reset clears the column data without releasing the backing array.
func (c *DateTimeColumn) Reset() { c.Data = c.Data[:0] }

// DecodeColumn decodes datetime rows from the wire protocol.
func (c *DateTimeColumn) DecodeColumn(r *proto.Reader, rows int) error {
	if rows == 0 {
		c.Data = c.Data[:0]
		return nil
	}
	c.Data = make([]uint32, rows)
	for i := range c.Data {
		v, err := r.UInt32()
		if err != nil {
			return err
		}
		c.Data[i] = v
	}
	return nil
}

// EncodeColumn encodes datetime data to the wire buffer.
func (c *DateTimeColumn) EncodeColumn(b *proto.Buffer) error {
	for _, v := range c.Data {
		b.PutUInt32(v)
	}
	return nil
}

// WriteColumn writes the datetime column to the wire writer.
func (c *DateTimeColumn) WriteColumn(w *proto.Writer) {
	w.ChainBuffer(func(b *proto.Buffer) {
		for _, v := range c.Data {
			b.PutUInt32(v)
		}
	})
}
