package column

import (
	"fmt"
	"strconv"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/shopspring/decimal"
)

type Decimal128Column struct {
	name      string
	Scale     int32
	precision int
	Data      []Int128
}

func NewDecimal128Column(name string) *Decimal128Column {
	return &Decimal128Column{name: name}
}

func (c *Decimal128Column) Name() string { return c.name }

func (c *Decimal128Column) Type() proto.ColumnType {
	return proto.ColumnTypeDecimal128.With(
		strconv.Itoa(c.precision),
		strconv.Itoa(int(c.Scale)),
	)
}

func (c *Decimal128Column) Len() int { return len(c.Data) }

func (c *Decimal128Column) Append(v decimal.Decimal) {
	bi := v.Shift(c.Scale).BigInt()
	iv, err := Int128FromBigInt(bi)
	if err != nil {
		panic(fmt.Sprintf("decimal128: %v", err))
	}
	c.Data = append(c.Data, iv)
}

func (c *Decimal128Column) AppendArr(v []decimal.Decimal) {
	for _, d := range v {
		c.Append(d)
	}
}

func (c *Decimal128Column) Row(i int) decimal.Decimal {
	bi := c.Data[i].ToBigInt()
	return decimal.NewFromBigInt(bi, -c.Scale)
}

func (c *Decimal128Column) Reset() { c.Data = c.Data[:0] }

func (c *Decimal128Column) Infer(t proto.ColumnType) error {
	base := t.Base()
	if base != proto.ColumnTypeDecimal128 && base != proto.ColumnTypeDecimal {
		return fmt.Errorf("decimal128: expected Decimal128 or Decimal, got %q", base)
	}
	prec, scale, err := parseDecimalParams(string(t.Elem()))
	if err != nil {
		return err
	}
	if prec > 38 {
		panic(fmt.Sprintf("decimal128: precision %d exceeds max 38", prec))
	}
	if prec < 1 {
		prec = 38
	}
	c.precision = prec
	c.Scale = int32(scale)
	return nil
}

func (c *Decimal128Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 16, &c.Data)
}

func (c *Decimal128Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 16, b)
}

func (c *Decimal128Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 16, w)
}
