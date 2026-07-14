package column

import (
	"time"
	"unsafe"

	"github.com/ClickHouse/ch-go/proto"
)

// DateColumn is a Date column (stored as days since epoch, uint16).
type DateColumn struct {
	name string
	Data []uint16
}

// NewDateColumn creates a DateColumn with the given column name.
func NewDateColumn(name string) *DateColumn {
	return &DateColumn{name: name}
}

// Name returns the column name.
func (c *DateColumn) Name() string { return c.name }

// Type returns proto.ColumnTypeDate.
func (c *DateColumn) Type() proto.ColumnType { return proto.ColumnTypeDate }

// Len returns the number of elements in the column.
func (c *DateColumn) Len() int { return len(c.Data) }

// Reset clears the column data without releasing the backing array.
func (c *DateColumn) Reset() { c.Data = c.Data[:0] }

// Append adds a single time value, stored as days since epoch.
func (c *DateColumn) Append(v time.Time) {
	c.Data = append(c.Data, uint16(v.Unix()/86400))
}

// AppendArr adds multiple time values.
func (c *DateColumn) AppendArr(vs []time.Time) {
	for _, v := range vs {
		c.Data = append(c.Data, uint16(v.Unix()/86400))
	}
}

// Row returns the time value at index i (UTC date).
func (c *DateColumn) Row(i int) time.Time {
	return time.Unix(int64(c.Data[i])*86400, 0).UTC()
}

// DecodeColumn decodes Date rows from the wire protocol.
func (c *DateColumn) DecodeColumn(r *proto.Reader, rows int) error {
	if rows == 0 {
		c.Data = c.Data[:0]
		return nil
	}
	c.Data = make([]uint16, rows)
	dest := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), rows*2)
	return r.ReadFull(dest)
}

// EncodeColumn encodes Date data to the wire buffer.
func (c *DateColumn) EncodeColumn(b *proto.Buffer) error {
	if len(c.Data) == 0 {
		return nil
	}
	off := len(b.Buf)
	byteLen := len(c.Data) * 2
	b.Buf = append(b.Buf, make([]byte, byteLen)...)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), byteLen)
	copy(b.Buf[off:], src)
	return nil
}

// WriteColumn writes the Date column to the wire writer.
func (c *DateColumn) WriteColumn(w *proto.Writer) {
	w.ChainBuffer(func(b *proto.Buffer) {
		if len(c.Data) == 0 {
			return
		}
		off := len(b.Buf)
		byteLen := len(c.Data) * 2
		b.Buf = append(b.Buf, make([]byte, byteLen)...)
		src := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), byteLen)
		copy(b.Buf[off:], src)
	})
}
