package column

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

// DateTime64Column is a DateTime64 column with configurable precision and timezone.
type DateTime64Column struct {
	name      string
	Data      []int64
	Precision proto.Precision
	Location  *time.Location
}

// NewDateTime64Column creates a DateTime64Column with the given column name and precision.
func NewDateTime64Column(name string, precision proto.Precision) *DateTime64Column {
	return &DateTime64Column{name: name, Precision: precision}
}

func (c *DateTime64Column) Name() string { return c.name }

func (c *DateTime64Column) Type() proto.ColumnType {
	var elems []string
	elems = append(elems, strconv.Itoa(int(c.Precision)))
	if loc := c.Location; loc != nil {
		elems = append(elems, fmt.Sprintf("'%s'", loc))
	}
	return proto.ColumnTypeDateTime64.With(elems...)
}

func (c *DateTime64Column) Len() int { return len(c.Data) }

func (c *DateTime64Column) Append(v time.Time) {
	c.Data = append(c.Data, v.UnixNano()/c.Precision.Scale())
}

func (c *DateTime64Column) AppendArr(vs []time.Time) {
	for _, v := range vs {
		c.Data = append(c.Data, v.UnixNano()/c.Precision.Scale())
	}
}

func (c *DateTime64Column) Row(i int) time.Time {
	nsec := c.Data[i] * c.Precision.Scale()
	t := time.Unix(nsec/1e9, nsec%1e9)
	if c.Location != nil {
		return t.In(c.Location)
	}
	return t.In(time.Local)
}

func (c *DateTime64Column) Reset() { c.Data = c.Data[:0] }

func (c *DateTime64Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 8, &c.Data)
}

func (c *DateTime64Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 8, b)
}

func (c *DateTime64Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 8, w)
}

// Infer sets Precision from a DateTime64 column type string (e.g. "DateTime64(3)" or "DateTime64(3, 'UTC')").
func (c *DateTime64Column) Infer(t proto.ColumnType) error {
	elem := string(t.Elem())
	if elem == "" {
		return fmt.Errorf("datetime64: no elements in %q", t)
	}
	pStr, locStr, hasloc := strings.Cut(elem, ",")
	pStr = strings.Trim(pStr, "' ")
	n, err := strconv.ParseUint(pStr, 10, 8)
	if err != nil {
		return fmt.Errorf("datetime64: parse precision: %w", err)
	}
	p := proto.Precision(n)
	if !p.Valid() {
		return fmt.Errorf("datetime64: precision %d out of range", n)
	}
	c.Precision = p
	if hasloc {
		locStr = strings.Trim(locStr, "' ")
		loc, err := time.LoadLocation(locStr)
		if err != nil {
			return fmt.Errorf("datetime64: invalid location %q: %w", locStr, err)
		}
		c.Location = loc
	}
	return nil
}
