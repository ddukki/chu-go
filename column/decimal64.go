package column

import (
	"fmt"
	"strconv"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/shopspring/decimal"
)

type Decimal64Column struct {
	name      string
	Scale     int32
	precision int
	Data      []int64
}

func NewDecimal64Column(name string) *Decimal64Column {
	return &Decimal64Column{name: name}
}

func (c *Decimal64Column) Name() string { return c.name }

func (c *Decimal64Column) Type() proto.ColumnType {
	return proto.ColumnTypeDecimal64.With(
		strconv.Itoa(c.precision),
		strconv.Itoa(int(c.Scale)),
	)
}

func (c *Decimal64Column) Len() int { return len(c.Data) }

func (c *Decimal64Column) Append(v decimal.Decimal) {
	val := v.Shift(c.Scale).IntPart()
	c.Data = append(c.Data, val)
}

func (c *Decimal64Column) AppendArr(v []decimal.Decimal) {
	for _, d := range v {
		c.Append(d)
	}
}

func (c *Decimal64Column) Row(i int) decimal.Decimal {
	return decimal.New(c.Data[i], -c.Scale)
}

func (c *Decimal64Column) Reset() { c.Data = c.Data[:0] }

func (c *Decimal64Column) Infer(t proto.ColumnType) error {
	base := t.Base()
	if base != proto.ColumnTypeDecimal64 && base != proto.ColumnTypeDecimal {
		return fmt.Errorf("decimal64: expected Decimal64 or Decimal, got %q", base)
	}
	prec, scale, err := parseDecimalParams(string(t.Elem()))
	if err != nil {
		return err
	}
	if prec > 18 {
		panic(fmt.Sprintf("decimal64: precision %d exceeds max 18", prec))
	}
	if prec < 1 {
		prec = 18
	}
	c.precision = prec
	c.Scale = int32(scale)
	return nil
}

func (c *Decimal64Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 8, &c.Data)
}

func (c *Decimal64Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 8, b)
}

func (c *Decimal64Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 8, w)
}
