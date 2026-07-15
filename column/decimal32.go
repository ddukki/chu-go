package column

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/shopspring/decimal"
)

type Decimal32Column struct {
	name      string
	Scale     int32
	precision int
	Data      []int32
}

func NewDecimal32Column(name string) *Decimal32Column {
	return &Decimal32Column{name: name}
}

func (c *Decimal32Column) Name() string { return c.name }

func (c *Decimal32Column) Type() proto.ColumnType {
	return proto.ColumnTypeDecimal32.With(
		strconv.Itoa(c.precision),
		strconv.Itoa(int(c.Scale)),
	)
}

func (c *Decimal32Column) Len() int { return len(c.Data) }

func (c *Decimal32Column) Append(v decimal.Decimal) {
	val := v.Shift(c.Scale).IntPart()
	c.Data = append(c.Data, int32(val))
}

func (c *Decimal32Column) AppendArr(v []decimal.Decimal) {
	for _, d := range v {
		c.Append(d)
	}
}

func (c *Decimal32Column) Row(i int) decimal.Decimal {
	return decimal.New(int64(c.Data[i]), -c.Scale)
}

func (c *Decimal32Column) Reset() { c.Data = c.Data[:0] }

func (c *Decimal32Column) Infer(t proto.ColumnType) error {
	base := t.Base()
	if base != proto.ColumnTypeDecimal32 && base != proto.ColumnTypeDecimal {
		return fmt.Errorf("decimal32: expected Decimal32 or Decimal, got %q", base)
	}
	prec, scale, err := parseDecimalParams(string(t.Elem()))
	if err != nil {
		return err
	}
	if prec > 9 {
		panic(fmt.Sprintf("decimal32: precision %d exceeds max 9", prec))
	}
	if prec < 1 {
		prec = 9
	}
	c.precision = prec
	c.Scale = int32(scale)
	return nil
}

func (c *Decimal32Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 4, &c.Data)
}

func (c *Decimal32Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 4, b)
}

func (c *Decimal32Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 4, w)
}

func parseDecimalParams(elem string) (precision, scale int, err error) {
	if elem == "" {
		return 0, 0, nil
	}
	parts := strings.SplitN(elem, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("decimal: expected precision,scale, got %q", elem)
	}
	prec, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("decimal: invalid precision %q: %w", parts[0], err)
	}
	s, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("decimal: invalid scale %q: %w", parts[1], err)
	}
	if s < 0 {
		return 0, 0, fmt.Errorf("decimal: scale %d cannot be negative", s)
	}
	if s > math.MaxInt32 {
		return 0, 0, fmt.Errorf("decimal: scale %d too large", s)
	}
	return prec, s, nil
}
