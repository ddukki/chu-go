package column

import (
	"net"
	"unsafe"

	"github.com/ClickHouse/ch-go/proto"
)

// IPv6 is a 16-byte IPv6 address.
type IPv6 [16]byte

// IPv6Column is an IPv6 column.
type IPv6Column struct {
	name string
	Data []IPv6
}

// NewIPv6Column creates an IPv6Column with the given column name.
func NewIPv6Column(name string) *IPv6Column {
	return &IPv6Column{name: name}
}

// Name returns the column name.
func (c *IPv6Column) Name() string { return c.name }

// Type returns proto.ColumnTypeIPv6.
func (c *IPv6Column) Type() proto.ColumnType { return proto.ColumnTypeIPv6 }

// Len returns the number of elements in the column.
func (c *IPv6Column) Len() int { return len(c.Data) }

// Reset clears the column data without releasing the backing array.
func (c *IPv6Column) Reset() { c.Data = c.Data[:0] }

// Append adds a single IPv6 value.
func (c *IPv6Column) Append(v IPv6) { c.Data = append(c.Data, v) }

// AppendArr adds multiple IPv6 values.
func (c *IPv6Column) AppendArr(vs []IPv6) { c.Data = append(c.Data, vs...) }

// Row returns the IPv6 at index i.
func (c *IPv6Column) Row(i int) IPv6 { return c.Data[i] }

// DecodeColumn decodes IPv6 rows from the wire protocol.
func (c *IPv6Column) DecodeColumn(r *proto.Reader, rows int) error {
	if rows == 0 {
		c.Data = c.Data[:0]
		return nil
	}
	c.Data = make([]IPv6, rows)
	dest := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), rows*16)
	return r.ReadFull(dest)
}

// EncodeColumn encodes IPv6 data to the wire buffer.
func (c *IPv6Column) EncodeColumn(b *proto.Buffer) error {
	if len(c.Data) == 0 {
		return nil
	}
	off := len(b.Buf)
	b.Buf = append(b.Buf, make([]byte, len(c.Data)*16)...)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), len(c.Data)*16)
	copy(b.Buf[off:], src)
	return nil
}

// WriteColumn writes the IPv6 column to the wire writer.
func (c *IPv6Column) WriteColumn(w *proto.Writer) {
	w.ChainBuffer(func(b *proto.Buffer) {
		if len(c.Data) == 0 {
			return
		}
		off := len(b.Buf)
		b.Buf = append(b.Buf, make([]byte, len(c.Data)*16)...)
		src := unsafe.Slice((*byte)(unsafe.Pointer(&c.Data[0])), len(c.Data)*16)
		copy(b.Buf[off:], src)
	})
}

// IPv6ToNet converts an IPv6 value to a net.IP.
func IPv6ToNet(v IPv6) net.IP { return net.IP(v[:]) }

// NetToIPv6 converts a net.IP to an IPv6 value. Returns false if the input
// is not a valid 16-byte IPv6 address.
func NetToIPv6(ip net.IP) (IPv6, bool) {
	if len(ip) != 16 {
		return IPv6{}, false
	}
	var v IPv6
	copy(v[:], ip)
	return v, true
}

// String returns the colon-hex format.
func (v IPv6) String() string { return net.IP(v[:]).String() }
