package column

import (
	"unsafe"

	"github.com/ClickHouse/ch-go/proto"
)

type Nullable[T any] struct {
	Values ColumnOf[T]
	Nulls  []bool
}

func NewNullable[T any](col ColumnOf[T]) *Nullable[T] {
	return &Nullable[T]{Values: col}
}

func (c *Nullable[T]) Name() string { return c.Values.Name() }

func (c *Nullable[T]) Type() proto.ColumnType {
	return proto.ColumnTypeNullable.Sub(c.Values.Type())
}

func (c *Nullable[T]) Len() int { return len(c.Nulls) }

func (c *Nullable[T]) Append(v T, isNull bool) {
	if isNull {
		var zero T
		c.Values.Append(zero)
	} else {
		c.Values.Append(v)
	}
	c.Nulls = append(c.Nulls, isNull)
}

func (c *Nullable[T]) AppendArr(v []T) {
	for _, x := range v {
		c.Append(x, false)
	}
}

func (c *Nullable[T]) Row(i int) (T, bool) {
	return c.Values.Row(i), c.Nulls[i]
}

func (c *Nullable[T]) DecodeColumn(r *proto.Reader, rows int) error {
	if cap(c.Nulls) >= rows {
		c.Nulls = c.Nulls[:rows]
	} else {
		c.Nulls = make([]bool, rows)
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&c.Nulls[0])), rows)
	if err := r.ReadFull(buf); err != nil {
		return err
	}
	return c.Values.DecodeColumn(r, rows)
}

func (c *Nullable[T]) EncodeColumn(b *proto.Buffer) error {
	for _, isNull := range c.Nulls {
		if isNull {
			b.Buf = append(b.Buf, 1)
		} else {
			b.Buf = append(b.Buf, 0)
		}
	}
	return c.Values.EncodeColumn(b)
}

func (c *Nullable[T]) WriteColumn(w *proto.Writer) {
	w.ChainBuffer(func(b *proto.Buffer) {
		for _, isNull := range c.Nulls {
			if isNull {
				b.Buf = append(b.Buf, 1)
			} else {
				b.Buf = append(b.Buf, 0)
			}
		}
	})
	c.Values.WriteColumn(w)
}
