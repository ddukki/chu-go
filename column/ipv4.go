package column

import (
	"net"
	"unsafe"

	"github.com/ClickHouse/ch-go/proto"
)

// IPv4 is a 4-byte IPv4 address.
type IPv4 [4]byte

// IPv4Column is an IPv4 column.
type IPv4Column struct {
	name string
	Data []IPv4
}

// NewIPv4Column creates an IPv4Column with the given column name.
func NewIPv4Column(name string) *IPv4Column {
	return &IPv4Column{name: name}
}

// Name returns the column name.
func (c *IPv4Column) Name() string { return c.name }

// Type returns proto.ColumnTypeIPv4.
func (c *IPv4Column) Type() proto.ColumnType { return proto.ColumnTypeIPv4 }

// Len returns the number of elements in the column.
func (c *IPv4Column) Len() int { return len(c.Data) }

// Reset clears the column data without releasing the backing array.
func (c *IPv4Column) Reset() { c.Data = c.Data[:0] }

// Append adds a single IPv4 value.
func (c *IPv4Column) Append(v IPv4) { c.Data = append(c.Data, v) }

// AppendArr adds multiple IPv4 values.
func (c *IPv4Column) AppendArr(vs []IPv4) { c.Data = append(c.Data, vs...) }

// Row returns the IPv4 at index i.
func (c *IPv4Column) Row(i int) IPv4 { return c.Data[i] }

// DecodeColumn decodes IPv4 rows from the wire protocol.
func (c *IPv4Column) DecodeColumn(r *proto.Reader, rows int) error {
	if rows == 0 {
		c.Data = c.Data[:0]
		return nil
	}
	c.Data = make([]IPv4, rows)
	dest := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), rows*4)
	return r.ReadFull(dest)
}

// EncodeColumn encodes IPv4 data to the wire buffer.
func (c *IPv4Column) EncodeColumn(b *proto.Buffer) error {
	if len(c.Data) == 0 {
		return nil
	}
	off := len(b.Buf)
	b.Buf = append(b.Buf, make([]byte, len(c.Data)*4)...)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), len(c.Data)*4)
	copy(b.Buf[off:], src)
	return nil
}

// WriteColumn writes the IPv4 column to the wire writer.
func (c *IPv4Column) WriteColumn(w *proto.Writer) {
	w.ChainBuffer(func(b *proto.Buffer) {
		if len(c.Data) == 0 {
			return
		}
		off := len(b.Buf)
		b.Buf = append(b.Buf, make([]byte, len(c.Data)*4)...)
		src := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), len(c.Data)*4)
		copy(b.Buf[off:], src)
	})
}

// IPv4ToNet converts an IPv4 value to a net.IP.
func IPv4ToNet(v IPv4) net.IP { return net.IP(v[:]) }

// NetToIPv4 converts a net.IP to an IPv4 value. It returns false if the input
// is not an IPv4 address (uses net.IP.To4() internally — accepts 4-byte and
// IPv4-mapped IPv6 addresses).
func NetToIPv4(ip net.IP) (IPv4, bool) {
	ip4 := ip.To4()
	if ip4 == nil {
		return IPv4{}, false
	}
	var v IPv4
	copy(v[:], ip4)
	return v, true
}

// String returns the dotted-decimal format.
func (v IPv4) String() string { return net.IP(v[:]).String() }
