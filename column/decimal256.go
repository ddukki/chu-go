package column

import (
	"fmt"
	"strconv"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/shopspring/decimal"
)

type Decimal256Column struct {
	name      string
	Scale     int32
	precision int
	Data      []Int256
}

func NewDecimal256Column(name string) *Decimal256Column {
	return &Decimal256Column{name: name}
}

func (c *Decimal256Column) Name() string { return c.name }

func (c *Decimal256Column) Type() proto.ColumnType {
	return proto.ColumnTypeDecimal256.With(
		strconv.Itoa(c.precision),
		strconv.Itoa(int(c.Scale)),
	)
}

func (c *Decimal256Column) Len() int { return len(c.Data) }

func (c *Decimal256Column) Append(v decimal.Decimal) {
	bi := v.Shift(c.Scale).BigInt()
	iv, err := Int256FromBigInt(bi)
	if err != nil {
		panic(fmt.Sprintf("decimal256: %v", err))
	}
	c.Data = append(c.Data, iv)
}

func (c *Decimal256Column) AppendArr(v []decimal.Decimal) {
	for _, d := range v {
		c.Append(d)
	}
}

func (c *Decimal256Column) Row(i int) decimal.Decimal {
	bi := c.Data[i].ToBigInt()
	return decimal.NewFromBigInt(bi, -c.Scale)
}

func (c *Decimal256Column) Reset() { c.Data = c.Data[:0] }

func (c *Decimal256Column) Infer(t proto.ColumnType) error {
	base := t.Base()
	if base != proto.ColumnTypeDecimal256 && base != proto.ColumnTypeDecimal {
		return fmt.Errorf("decimal256: expected Decimal256 or Decimal, got %q", base)
	}
	prec, scale, err := parseDecimalParams(string(t.Elem()))
	if err != nil {
		return err
	}
	if prec > 76 {
		panic(fmt.Sprintf("decimal256: precision %d exceeds max 76", prec))
	}
	if prec < 1 {
		prec = 76
	}
	c.precision = prec
	c.Scale = int32(scale)
	return nil
}

func (c *Decimal256Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 32, &c.Data)
}

func (c *Decimal256Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 32, b)
}

func (c *Decimal256Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 32, w)
}
