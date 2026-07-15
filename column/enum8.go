package column

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ClickHouse/ch-go/proto"
)

// Enum8Column is an Enum8 column storing raw int8 values with string mapping.
type Enum8Column struct {
	name     string
	t        proto.ColumnType
	rawToStr map[int8]string
	strToRaw map[string]int8
	Data     []int8
}

func NewEnum8Column(name string) *Enum8Column {
	return &Enum8Column{name: name}
}

func (c *Enum8Column) Name() string { return c.name }

func (c *Enum8Column) Type() proto.ColumnType {
	if c.t != "" {
		return c.t
	}
	return proto.ColumnTypeEnum8
}

func (c *Enum8Column) Len() int { return len(c.Data) }

func (c *Enum8Column) Row(i int) string {
	v := c.Data[i]
	if s, ok := c.rawToStr[v]; ok {
		return s
	}
	return strconv.Itoa(int(v))
}

func (c *Enum8Column) Append(v string) {
	if v == "" {
		c.Data = append(c.Data, 0)
		return
	}
	n, ok := c.strToRaw[v]
	if !ok {
		panic(fmt.Sprintf("column: Enum8Column: unknown name %q", v))
	}
	c.Data = append(c.Data, n)
}

func (c *Enum8Column) AppendArr(v []string) {
	for _, vv := range v {
		c.Append(vv)
	}
}

func (c *Enum8Column) Reset() { c.Data = c.Data[:0] }

func (c *Enum8Column) DecodeColumn(r *proto.Reader, rows int) error {
	return decodeFixed(r, rows, 1, &c.Data)
}

func (c *Enum8Column) EncodeColumn(b *proto.Buffer) error {
	return encodeFixed(c.Data, 1, b)
}

func (c *Enum8Column) WriteColumn(w *proto.Writer) {
	writeFixed(c.Data, 1, w)
}

func (c *Enum8Column) Infer(t proto.ColumnType) error {
	pairs, err := parseEnumType(t)
	if err != nil {
		return err
	}
	if t.Base() != proto.ColumnTypeEnum8 {
		return fmt.Errorf("enum: expected Enum8, got %q", t.Base())
	}
	c.rawToStr = make(map[int8]string, len(pairs))
	c.strToRaw = make(map[string]int8, len(pairs))
	for _, p := range pairs {
		if p.value < -128 || p.value > 127 {
			return fmt.Errorf("enum: value %d out of range for Enum8", p.value)
		}
		v := int8(p.value)
		c.rawToStr[v] = p.name
		c.strToRaw[p.name] = v
	}
	c.t = t
	return nil
}

// parseEnumType parses a column type string like "Enum8('a'=1, 'b'=2)" or
// "Enum16('x'=100, 'y'=200)" into a list of name-value pairs.
func parseEnumType(t proto.ColumnType) ([]struct {
	name  string
	value int
}, error) {
	elem := string(t.Elem())
	if elem == "" {
		return nil, fmt.Errorf("enum: no elements in %q", t)
	}
	var pairs []struct {
		name  string
		value int
	}
	for _, part := range strings.Split(elem, ",") {
		part = strings.TrimSpace(part)
		left, right, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("enum: bad element %q", part)
		}
		left = strings.TrimSpace(left)
		left = strings.Trim(left, "'")
		right = strings.TrimSpace(right)
		n, err := strconv.Atoi(right)
		if err != nil {
			return nil, fmt.Errorf("enum: bad value %q: %w", right, err)
		}
		pairs = append(pairs, struct {
			name  string
			value int
		}{left, n})
	}
	return pairs, nil
}
