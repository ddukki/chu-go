package column

import (
	"bytes"
	"math"
	"net"
	"testing"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/google/uuid"
)

func TestUUIDColumnRoundTrip(t *testing.T) {
	u1, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
	u2, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	u3, _ := uuid.Parse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	col := NewUUIDColumn("id")
	col.Append(UUID(u1))
	col.Append(UUID(u2))
	col.Append(UUID(u3))

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewUUIDColumn("id")
	if err := got.DecodeColumn(r, 3); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 3 {
		t.Fatalf("len: got %d, want 3", got.Len())
	}

	for i, want := range []UUID{UUID(u1), UUID(u2), UUID(u3)} {
		if got.Row(i) != want {
			t.Fatalf("row %d: got %v, want %v", i, got.Row(i), want)
		}
	}
}

func TestUUIDColumnZeroRows(t *testing.T) {
	col := NewUUIDColumn("id")
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewUUIDColumn("id")
	if err := got.DecodeColumn(r, 0); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 0 {
		t.Fatalf("len: got %d, want 0", got.Len())
	}
}

func TestUUIDColumnReset(t *testing.T) {
	col := NewUUIDColumn("id")
	var v UUID
	v[15] = 1
	col.Append(v)
	col.Reset()
	if col.Len() != 0 {
		t.Fatal("Reset should clear data")
	}
	col.Append(v)
	if col.Len() != 1 {
		t.Fatal("Reset should preserve capacity for re-use")
	}
}

func TestUUIDConversions(t *testing.T) {
	want, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	got := GoToUUID(want)
	back := UUIDToGo(got)
	if want != back {
		t.Fatalf("UUID roundtrip: got %v, want %v", back, want)
	}
}

func TestUUIDString(t *testing.T) {
	u, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	v := UUID(u)
	if v.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("UUID.String(): got %q", v.String())
	}
}

func TestIPv4ColumnRoundTrip(t *testing.T) {
	col := NewIPv4Column("ip")
	col.Append(IPv4{10, 0, 0, 1})
	col.Append(IPv4{192, 168, 1, 1})
	col.Append(IPv4{255, 255, 255, 255})

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewIPv4Column("ip")
	if err := got.DecodeColumn(r, 3); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 3 {
		t.Fatalf("len: got %d, want 3", got.Len())
	}

	wants := []IPv4{{10, 0, 0, 1}, {192, 168, 1, 1}, {255, 255, 255, 255}}
	for i, want := range wants {
		if got.Row(i) != want {
			t.Fatalf("row %d: got %v, want %v", i, got.Row(i), want)
		}
	}
}

func TestIPv4Conversions(t *testing.T) {
	v := IPv4{192, 168, 1, 1}
	ip := IPv4ToNet(v)
	if ip.String() != "192.168.1.1" {
		t.Fatalf("IPv4ToNet: got %s", ip.String())
	}

	back, ok := NetToIPv4(ip)
	if !ok {
		t.Fatal("NetToIPv4 should succeed for 4-byte input")
	}
	if back != v {
		t.Fatalf("NetToIPv4 roundtrip: got %v, want %v", back, v)
	}
}

func TestNetToIPv4Mapped(t *testing.T) {
	mapped := net.IPv4(10, 0, 0, 1).To16()
	v, ok := NetToIPv4(mapped)
	if !ok {
		t.Fatal("NetToIPv4 should accept IPv4-mapped IPv6")
	}
	if v != (IPv4{10, 0, 0, 1}) {
		t.Fatalf("NetToIPv4 mapped: got %v", v)
	}
}

func TestNetToIPv4Rejects(t *testing.T) {
	if _, ok := NetToIPv4(nil); ok {
		t.Fatal("NetToIPv4 should reject nil")
	}
	if _, ok := NetToIPv4(net.ParseIP("::1")); ok {
		t.Fatal("NetToIPv4 should reject normal IPv6")
	}
}

func TestIPv4String(t *testing.T) {
	v := IPv4{192, 168, 1, 1}
	if v.String() != "192.168.1.1" {
		t.Fatalf("IPv4.String(): got %q", v.String())
	}
}

func TestIPv6ColumnRoundTrip(t *testing.T) {
	col := NewIPv6Column("ip6")
	col.Append(IPv6{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	col.Append(IPv6{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	col.Append(IPv6{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewIPv6Column("ip6")
	if err := got.DecodeColumn(r, 3); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 3 {
		t.Fatalf("len: got %d, want 3", got.Len())
	}
}

func TestIPv6Conversions(t *testing.T) {
	v := IPv6{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	ip := IPv6ToNet(v)
	if !ip.Equal(net.ParseIP("2001:db8::1")) {
		t.Fatalf("IPv6ToNet: got %s", ip.String())
	}

	back, ok := NetToIPv6(ip)
	if !ok {
		t.Fatal("NetToIPv6 should succeed for 16-byte input")
	}
	if back != v {
		t.Fatalf("NetToIPv6 roundtrip: got %v, want %v", back, v)
	}
}

func TestNetToIPv6Rejects(t *testing.T) {
	if _, ok := NetToIPv6(nil); ok {
		t.Fatal("NetToIPv6 should reject nil")
	}
	if _, ok := NetToIPv6(net.IP{192, 168, 1, 1}); ok {
		t.Fatal("NetToIPv6 should reject 4-byte input")
	}
}

func TestIPv6String(t *testing.T) {
	v := IPv6{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	if v.String() != "2001:db8::1" {
		t.Fatalf("IPv6.String(): got %q", v.String())
	}
}

func TestDateColumnRoundTrip(t *testing.T) {
	t1 := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)

	col := NewDateColumn("d")
	col.Append(t1)
	col.Append(t2)
	col.Append(t3)

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewDateColumn("d")
	if err := got.DecodeColumn(r, 3); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 3 {
		t.Fatalf("len: got %d, want 3", got.Len())
	}

	wants := []time.Time{t1, t2, t3}
	for i, want := range wants {
		if !got.Row(i).Equal(want) {
			t.Fatalf("row %d: got %v, want %v", i, got.Row(i), want)
		}
	}
}

func TestDateColumnAppendStoresDays(t *testing.T) {
	col := NewDateColumn("d")
	col.Append(time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC))
	if col.Data[0] != uint16(time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC).Unix()/86400) {
		t.Fatal("Append should store days since epoch")
	}
}

func TestDateColumnZeroRows(t *testing.T) {
	col := NewDateColumn("d")
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewDateColumn("d")
	if err := got.DecodeColumn(r, 0); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 0 {
		t.Fatalf("len: got %d, want 0", got.Len())
	}
}

func TestDateColumnReset(t *testing.T) {
	col := NewDateColumn("d")
	col.Append(time.Now())
	col.Reset()
	if col.Len() != 0 {
		t.Fatal("Reset should clear data")
	}
}

func TestDateRowArithmetic(t *testing.T) {
	col := NewDateColumn("d")
	col.Data = []uint16{0, 1, 365, math.MaxUint16}

	wants := []time.Time{
		time.Unix(0, 0),
		time.Unix(86400, 0),
		time.Unix(365*86400, 0),
		time.Unix(int64(math.MaxUint16)*86400, 0),
	}
	for i, want := range wants {
		if !col.Row(i).Equal(want) {
			t.Fatalf("row %d: got %v, want %v", i, col.Row(i), want)
		}
	}
}

func TestDateTimeColumnReset(t *testing.T) {
	col := NewDateTimeColumn("ts")
	col.Append(time.Now())
	col.Reset()
	if col.Len() != 0 {
		t.Fatal("DateTimeColumn.Reset should clear data")
	}
}
