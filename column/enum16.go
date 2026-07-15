package column

import (
	"fmt"
	"strconv"

	"github.com/ClickHouse/ch-go/proto"
)

// Enum16Column is an Enum16 column storing raw int16 values with string mapping.
type Enum16Column struct {
	name     string
	t        proto.ColumnType
	rawToStr map[int16]string
	strToRaw map[string]int16
	Data     []int16
}

func NewEnum16Column(name string) *Enum16Column {
	return &Enum16Column{name: name}
}

func (c *Enum16Column) Name() string { return c.name }

func (c *Enum16Column) Type() proto.ColumnType {
	if c.t != "" {
		return c.t
	}
	return proto.ColumnTypeEnum16
}

func (c *Enum16Column) Len() int { return len(c.Data) }

func (c *Enum16Column) Row(i int) string {
	v := c.Data[i]
	if s, ok := c.rawToStr[v]; ok {
		return s
	}
	return strconv.Itoa(int(v))
}

func (c *Enum16Column) Append(v string) {
	if v == "" {
		c.Data = append(c.Data, 0)
		return
	}
	n, ok := c.strToRaw[v]
	if !ok {
		panic(fmt.Sprintf("column: Enum16Column: unknown name %q", v))
	}
	c.Data = append(c.Data, n)
}

func (c *Enum16Column) AppendArr(v []string) {
	for _, vv := range v {
		c.Append(vv)
	}
}

func (c *Enum16Column) Reset() { c.Data = c.Data[:0] }

func (c *Enum16Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 2, &c.Data)
}

func (c *Enum16Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 2, b)
}

func (c *Enum16Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 2, w)
}

func (c *Enum16Column) Infer(t proto.ColumnType) error {
	pairs, err := parseEnumType(t)
	if err != nil {
		return err
	}
	if t.Base() != proto.ColumnTypeEnum16 {
		return fmt.Errorf("enum: expected Enum16, got %q", t.Base())
	}
	c.rawToStr = make(map[int16]string, len(pairs))
	c.strToRaw = make(map[string]int16, len(pairs))
	for _, p := range pairs {
		if p.value < -32768 || p.value > 32767 {
			return fmt.Errorf("enum: value %d out of range for Enum16", p.value)
		}
		v := int16(p.value)
		c.rawToStr[v] = p.name
		c.strToRaw[p.name] = v
	}
	c.t = t
	return nil
}
