package column

import (
	"bytes"
	"math/big"
	"testing"
	"unsafe"

	"github.com/ClickHouse/ch-go/proto"
)

func TestInt128RoundTrip(t *testing.T) {
	vals := []Int128{
		{Lo: 0, Hi: 0},
		{Lo: 1, Hi: 0},
		{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF},
		{Lo: 0, Hi: 1},
		{Lo: 0xABCDEF0123456789, Hi: 0xDEADBEEFCAFEBABE},
	}
	col := NewInt128Column("test")
	for _, v := range vals {
		col.Append(v)
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewInt128Column("test")
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(vals)); err != nil {
		t.Fatal(err)
	}
	if got.Len() != len(vals) {
		t.Fatalf("Len: got %d, want %d", got.Len(), len(vals))
	}
	for i, v := range vals {
		if got.Row(i) != v {
			t.Fatalf("Row(%d): got %v, want %v", i, got.Row(i), v)
		}
	}
}

func TestInt128String(t *testing.T) {
	tests := []struct {
		v    Int128
		want string
	}{
		{Int128{Lo: 0, Hi: 0}, "0x00000000000000000000000000000000"},
		{Int128{Lo: 1, Hi: 0}, "0x00000000000000000000000000000001"},
		{Int128{Lo: 0, Hi: 1}, "0x00000000000000010000000000000000"},
		{Int128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}, "-0x00000000000000000000000000000001"},
	}
	for _, tc := range tests {
		got := tc.v.String()
		if got != tc.want {
			t.Errorf("Int128{%#x, %#x}.String() = %q, want %q", tc.v.Lo, tc.v.Hi, got, tc.want)
		}
	}
}

func TestInt128Cmp(t *testing.T) {
	z := Int128{0, 0}
	p1 := Int128{1, 0}
	n1 := Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}
	p2_64 := Int128{0, 1}          // 2^64
	n2_64 := Int128{0, 0xFFFFFFFFFFFFFFFF} // -(2^64)
	maxPos := Int128{0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF} // 2^127-1
	minNeg := Int128{0, 0x8000000000000000} // -2^127
	negMaxPos := Int128{1, 0x8000000000000000} // -(2^127-1)
	adj := Int128{2, 0}

	tests := []struct {
		a, b Int128
		want int
	}{
		// identity
		{z, z, 0}, {p1, p1, 0}, {n1, n1, 0}, {maxPos, maxPos, 0}, {minNeg, minNeg, 0},
		// zero vs signed
		{p1, z, 1}, {z, p1, -1},
		{n1, z, -1}, {z, n1, 1},
		// positive vs negative
		{p1, n1, 1}, {n1, p1, -1},
		{p2_64, n2_64, 1}, {n2_64, p2_64, -1},
		{maxPos, n1, 1}, {n1, maxPos, -1},
		{maxPos, minNeg, 1}, {minNeg, maxPos, -1},
		// magnitude — same sign
		{p2_64, p1, 1}, {p1, p2_64, -1},
		{maxPos, p2_64, 1}, {p2_64, maxPos, -1},
		{n2_64, n1, -1}, {n1, n2_64, 1},
		{minNeg, n2_64, -1}, {n2_64, minNeg, 1},
		{minNeg, negMaxPos, -1}, {negMaxPos, minNeg, 1},
		// adjacent
		{p1, z, 1}, {z, p1, -1},
		{adj, p1, 1}, {p1, adj, -1},
		{maxPos, negMaxPos, 1}, {negMaxPos, maxPos, -1},
	}
	for _, tc := range tests {
		got := tc.a.Cmp(tc.b)
		if got != tc.want {
			t.Errorf("Int128{%#x,%#x}.Cmp(Int128{%#x,%#x}) = %d, want %d",
				tc.a.Lo, tc.a.Hi, tc.b.Lo, tc.b.Hi, got, tc.want)
		}
	}
}

func TestInt128BigIntRoundTrip(t *testing.T) {
	vals := []Int128{
		{Lo: 0, Hi: 0},                                      // 0
		{Lo: 1, Hi: 0},                                      // 1
		{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF},    // -1
		{Lo: 0, Hi: 0x8000000000000000},                      // -2^127 (min)
		{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0x7FFFFFFFFFFFFFFF},    // 2^127-1 (max)
		{Lo: 0, Hi: 1},                                      // 2^64
		{Lo: 0, Hi: 0xFFFFFFFFFFFFFFFF},                      // -(2^64)
		{Lo: 1, Hi: 0x8000000000000000},                      // -(2^127-1)
		{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0},                      // 2^64-1
	}
	for _, v := range vals {
		bi := v.ToBigInt()
		got, err := Int128FromBigInt(bi)
		if err != nil {
			t.Fatalf("Int128FromBigInt(%v): %v", bi, err)
		}
		if got != v {
			t.Errorf("ToBigInt→FromBigInt: %v -> %v -> %v", v, bi, got)
		}
	}

	// Reverse direction: FromBigInt → ToBigInt
	bigVals := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(-1),
		new(big.Int).Lsh(big.NewInt(1), 64),
		new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 64)),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1)),  // 2^127-1
		new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 127)),                // -2^127
		big.NewInt(42),
		big.NewInt(-42),
	}
	for _, bi := range bigVals {
		v, err := Int128FromBigInt(bi)
		if err != nil {
			t.Fatalf("Int128FromBigInt(%v): %v", bi, err)
		}
		got := v.ToBigInt()
		if got.Cmp(bi) != 0 {
			t.Errorf("FromBigInt→ToBigInt: %v -> %v -> %v", bi, v, got)
		}
	}
}

func TestInt128ZeroRows(t *testing.T) {
	col := NewInt128Column("test")
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewInt128Column("test")
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 0); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 0 {
		t.Fatalf("Len: got %d, want 0", got.Len())
	}
}

func TestUInt128RoundTrip(t *testing.T) {
	vals := []UInt128{
		{Lo: 0, Hi: 0},
		{Lo: 1, Hi: 0},
		{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF},
		{Lo: 0, Hi: 1},
	}
	col := NewUInt128Column("test")
	for _, v := range vals {
		col.Append(v)
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewUInt128Column("test")
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(vals)); err != nil {
		t.Fatal(err)
	}
	for i, v := range vals {
		if got.Row(i) != v {
			t.Fatalf("Row(%d): got %v, want %v", i, got.Row(i), v)
		}
	}
}

func TestUInt128String(t *testing.T) {
	tests := []struct {
		v    UInt128
		want string
	}{
		{UInt128{Lo: 0, Hi: 0}, "0x00000000000000000000000000000000"},
		{UInt128{Lo: 1, Hi: 0}, "0x00000000000000000000000000000001"},
		{UInt128{Lo: 0, Hi: 1}, "0x00000000000000010000000000000000"},
	}
	for _, tc := range tests {
		got := tc.v.String()
		if got != tc.want {
			t.Errorf("UInt128{%#x, %#x}.String() = %q, want %q", tc.v.Lo, tc.v.Hi, got, tc.want)
		}
	}
}

func TestUInt128Cmp(t *testing.T) {
	z := UInt128{0, 0}
	p1 := UInt128{1, 0}
	p2_64 := UInt128{0, 1}
	half := UInt128{0, 0x8000000000000000}
	max := UInt128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}

	tests := []struct {
		a, b UInt128
		want int
	}{
		// identity
		{z, z, 0}, {p1, p1, 0}, {max, max, 0},
		// zero vs non-zero
		{p1, z, 1}, {z, p1, -1},
		{max, z, 1}, {z, max, -1},
		// magnitude
		{p2_64, p1, 1}, {p1, p2_64, -1},
		{half, p2_64, 1}, {p2_64, half, -1},
		{max, half, 1}, {half, max, -1},
		// adjacent
		{UInt128{2, 0}, p1, 1}, {p1, UInt128{2, 0}, -1},
		{UInt128{0xFFFFFFFFFFFFFFFF, 0}, UInt128{0xFFFFFFFFFFFFFFFE, 0}, 1},
	}
	for _, tc := range tests {
		got := tc.a.Cmp(tc.b)
		if got != tc.want {
			t.Errorf("UInt128{%#x,%#x}.Cmp(UInt128{%#x,%#x}) = %d, want %d",
				tc.a.Lo, tc.a.Hi, tc.b.Lo, tc.b.Hi, got, tc.want)
		}
	}
}

func TestUInt128BigIntRoundTrip(t *testing.T) {
	vals := []UInt128{
		{Lo: 0, Hi: 0},
		{Lo: 1, Hi: 0},
		{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}, // max
		{Lo: 0, Hi: 0x8000000000000000},
		{Lo: 0, Hi: 1},
		{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0},
	}
	for _, v := range vals {
		bi := v.ToBigInt()
		got, err := UInt128FromBigInt(bi)
		if err != nil {
			t.Fatalf("UInt128FromBigInt(%v): %v", bi, err)
		}
		if got != v {
			t.Errorf("ToBigInt→FromBigInt: %v -> %v -> %v", v, bi, got)
		}
	}

	bigVals := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		new(big.Int).Lsh(big.NewInt(1), 64),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)), // 2^128-1
		big.NewInt(42),
	}
	for _, bi := range bigVals {
		v, err := UInt128FromBigInt(bi)
		if err != nil {
			t.Fatalf("UInt128FromBigInt(%v): %v", bi, err)
		}
		got := v.ToBigInt()
		if got.Cmp(bi) != 0 {
			t.Errorf("FromBigInt→ToBigInt: %v -> %v -> %v", bi, v, got)
		}
	}
}

func TestInt256RoundTrip(t *testing.T) {
	vals := []Int256{
		{Lo: Int128{Lo: 0, Hi: 0}, Hi: Int128{Lo: 0, Hi: 0}},
		{Lo: Int128{Lo: 1, Hi: 0}, Hi: Int128{Lo: 0, Hi: 0}},
		{Lo: Int128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}, Hi: Int128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}},
		{Lo: Int128{Lo: 0, Hi: 0}, Hi: Int128{Lo: 1, Hi: 0}},
	}
	col := NewInt256Column("test")
	for _, v := range vals {
		col.Append(v)
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewInt256Column("test")
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(vals)); err != nil {
		t.Fatal(err)
	}
	for i, v := range vals {
		if got.Row(i) != v {
			t.Fatalf("Row(%d): got %v, want %v", i, got.Row(i), v)
		}
	}
}

func TestInt256String(t *testing.T) {
	tests := []struct {
		v    Int256
		want string
	}{
		{Int256{}, "0x0000000000000000000000000000000000000000000000000000000000000000"},
		{Int256{Lo: Int128{Lo: 1, Hi: 0}}, "0x0000000000000000000000000000000000000000000000000000000000000001"},
		{Int256{Lo: Int128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}, Hi: Int128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}}, "-0x0000000000000000000000000000000000000000000000000000000000000001"},
		{Int256{Lo: Int128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0}, Hi: Int128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}}, "-0x00000000000000000000000000000000ffffffffffffffff0000000000000001"},
	}
	for _, tc := range tests {
		got := tc.v.String()
		if got != tc.want {
			t.Errorf("Int256.String() = %q, want %q", got, tc.want)
		}
	}
}

func TestInt256Cmp(t *testing.T) {
	z := Int256{}
	p1 := Int256{Lo: Int128{1, 0}}
	negOne := Int256{
		Lo: Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		Hi: Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
	}
	p2_128 := Int256{Hi: Int128{1, 0}}
	n2_128 := Int256{Hi: Int128{0, 0xFFFFFFFFFFFFFFFF}}
	maxPos := Int256{
		Lo: Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		Hi: Int128{0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
	}
	minNeg := Int256{Hi: Int128{0, 0x8000000000000000}}
	maxNeg := Int256{Lo: Int128{1, 0}, Hi: Int128{0, 0x8000000000000000}}

	tests := []struct {
		a, b Int256
		want int
	}{
		// identity
		{z, z, 0}, {p1, p1, 0}, {negOne, negOne, 0}, {maxPos, maxPos, 0}, {minNeg, minNeg, 0},
		// zero vs signed
		{p1, z, 1}, {z, p1, -1},
		{negOne, z, -1}, {z, negOne, 1},
		// positive vs negative
		{p1, negOne, 1}, {negOne, p1, -1},
		{p2_128, n2_128, 1}, {n2_128, p2_128, -1},
		{maxPos, negOne, 1}, {negOne, maxPos, -1},
		{maxPos, minNeg, 1}, {minNeg, maxPos, -1},
		// magnitude — same sign
		{p2_128, p1, 1}, {p1, p2_128, -1},
		{maxPos, p2_128, 1}, {p2_128, maxPos, -1},
		{n2_128, negOne, -1}, {negOne, n2_128, 1},
		{minNeg, n2_128, -1}, {n2_128, minNeg, 1},
		{minNeg, maxNeg, -1}, {maxNeg, minNeg, 1},
		// adjacent
		{Int256{Lo: Int128{2, 0}}, p1, 1}, {p1, Int256{Lo: Int128{2, 0}}, -1},
	}
	for _, tc := range tests {
		got := tc.a.Cmp(tc.b)
		if got != tc.want {
			t.Errorf("Int256{%#x,%#x,%#x,%#x}.Cmp(...) = %d, want %d",
				tc.a.Lo.Lo, tc.a.Lo.Hi, tc.a.Hi.Lo, tc.a.Hi.Hi, got, tc.want)
		}
	}
}

func TestInt256BigIntRoundTrip(t *testing.T) {
	vals := []Int256{
		{},
		{Lo: Int128{1, 0}},
		{
			Lo: Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			Hi: Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
		}, // -1
		{Hi: Int128{0, 0x8000000000000000}}, // -2^255 (min)
		{
			Lo: Int128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF},
			Hi: Int128{0xFFFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF},
		}, // 2^255-1 (max)
		{Hi: Int128{1, 0}},                    // 2^128
		{Hi: Int128{0, 0xFFFFFFFFFFFFFFFF}},    // -(2^128)
	}
	for _, v := range vals {
		bi := v.ToBigInt()
		got, err := Int256FromBigInt(bi)
		if err != nil {
			t.Fatalf("Int256FromBigInt(%v): %v", bi, err)
		}
		if got != v {
			t.Errorf("ToBigInt→FromBigInt: %v -> %v -> %v", v, bi, got)
		}
	}

	bigVals := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(-1),
		new(big.Int).Lsh(big.NewInt(1), 128),
		new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 128)),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1)),  // 2^255-1
		new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255)),                // -2^255
		big.NewInt(42),
		big.NewInt(-42),
	}
	for _, bi := range bigVals {
		v, err := Int256FromBigInt(bi)
		if err != nil {
			t.Fatalf("Int256FromBigInt(%v): %v", bi, err)
		}
		got := v.ToBigInt()
		if got.Cmp(bi) != 0 {
			t.Errorf("FromBigInt→ToBigInt: %v -> %v -> %v", bi, v, got)
		}
	}
}

func TestUInt256BigIntRoundTrip(t *testing.T) {
	vals := []UInt256{
		{},
		{Lo: UInt128{1, 0}},
		{Lo: UInt128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, Hi: UInt128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
		{Hi: UInt128{1, 0}},
	}
	for _, v := range vals {
		bi := v.ToBigInt()
		got, err := UInt256FromBigInt(bi)
		if err != nil {
			t.Fatalf("UInt256FromBigInt(%v): %v", bi, err)
		}
		if got != v {
			t.Errorf("ToBigInt→FromBigInt: %v -> %v -> %v", v, bi, got)
		}
	}

	bigVals := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		new(big.Int).Lsh(big.NewInt(1), 128),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), // 2^256-1
		big.NewInt(42),
	}
	for _, bi := range bigVals {
		v, err := UInt256FromBigInt(bi)
		if err != nil {
			t.Fatalf("UInt256FromBigInt(%v): %v", bi, err)
		}
		got := v.ToBigInt()
		if got.Cmp(bi) != 0 {
			t.Errorf("FromBigInt→ToBigInt: %v -> %v -> %v", bi, v, got)
		}
	}
}

func TestUInt256Cmp(t *testing.T) {
	z := UInt256{}
	p1 := UInt256{Lo: UInt128{1, 0}}
	p2_128 := UInt256{Hi: UInt128{1, 0}}
	half := UInt256{Hi: UInt128{0, 0x8000000000000000}}
	max := UInt256{Lo: UInt128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, Hi: UInt128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}}

	tests := []struct {
		a, b UInt256
		want int
	}{
		{z, z, 0}, {p1, p1, 0}, {max, max, 0},
		{p1, z, 1}, {z, p1, -1},
		{max, z, 1}, {z, max, -1},
		{p2_128, p1, 1}, {p1, p2_128, -1},
		{half, p2_128, 1}, {p2_128, half, -1},
		{max, half, 1}, {half, max, -1},
		{UInt256{Lo: UInt128{2, 0}}, p1, 1}, {p1, UInt256{Lo: UInt128{2, 0}}, -1},
	}
	for _, tc := range tests {
		got := tc.a.Cmp(tc.b)
		if got != tc.want {
			t.Errorf("UInt256{%#x,%#x,%#x,%#x}.Cmp(...) = %d, want %d",
				tc.a.Lo.Lo, tc.a.Lo.Hi, tc.a.Hi.Lo, tc.a.Hi.Hi, got, tc.want)
		}
	}
}

func TestUInt256RoundTrip(t *testing.T) {
	vals := []UInt256{
		{},
		{Lo: UInt128{Lo: 1, Hi: 0}},
		{Lo: UInt128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}, Hi: UInt128{Lo: 0xFFFFFFFFFFFFFFFF, Hi: 0xFFFFFFFFFFFFFFFF}},
	}
	col := NewUInt256Column("test")
	for _, v := range vals {
		col.Append(v)
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewUInt256Column("test")
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(vals)); err != nil {
		t.Fatal(err)
	}
	for i, v := range vals {
		if got.Row(i) != v {
			t.Fatalf("Row(%d): got %v, want %v", i, got.Row(i), v)
		}
	}
}

func TestWideIntUnsafeSize(t *testing.T) {
	if sz := unsafe.Sizeof(Int128{}); sz != 16 {
		t.Fatalf("unsafe.Sizeof(Int128{}) = %d, want 16", sz)
	}
	if sz := unsafe.Sizeof(UInt128{}); sz != 16 {
		t.Fatalf("unsafe.Sizeof(UInt128{}) = %d, want 16", sz)
	}
	if sz := unsafe.Sizeof(Int256{}); sz != 32 {
		t.Fatalf("unsafe.Sizeof(Int256{}) = %d, want 32", sz)
	}
	if sz := unsafe.Sizeof(UInt256{}); sz != 32 {
		t.Fatalf("unsafe.Sizeof(UInt256{}) = %d, want 32", sz)
	}
}

func TestNullableInt128RoundTrip(t *testing.T) {
	inner := NewInt128Column("v")
	col := NewNullable(inner)
	vals := []Int128{
		{Lo: 0, Hi: 0},
		{Lo: 1, Hi: 0},
	}
	nulls := []bool{false, true}
	for i, v := range vals {
		col.Append(v, nulls[i])
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewNullable(NewInt128Column("v"))
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(vals)); err != nil {
		t.Fatal(err)
	}
	for i, v := range vals {
		r, isNull := got.Row(i)
		if isNull != nulls[i] {
			t.Fatalf("Row(%d) null: got %v, want %v", i, isNull, nulls[i])
		}
		if !isNull && r != v {
			t.Fatalf("Row(%d): got %v, want %v", i, r, v)
		}
	}
}

func TestInt128AppendArr(t *testing.T) {
	vals := []Int128{{Lo: 1, Hi: 0}, {Lo: 2, Hi: 0}, {Lo: 3, Hi: 0}}
	col := NewInt128Column("test")
	col.AppendArr(vals)
	if col.Len() != 3 {
		t.Fatalf("Len: got %d, want 3", col.Len())
	}
	for i, v := range vals {
		if col.Row(i) != v {
			t.Fatalf("Row(%d): got %v, want %v", i, col.Row(i), v)
		}
	}
}

func TestInt128Reset(t *testing.T) {
	col := NewInt128Column("test")
	col.Append(Int128{Lo: 1, Hi: 0})
	col.Reset()
	if col.Len() != 0 {
		t.Fatalf("Len after Reset: got %d, want 0", col.Len())
	}
}

func TestInt128Name(t *testing.T) {
	col := NewInt128Column("my_int128")
	if col.Name() != "my_int128" {
		t.Fatalf("Name: got %q, want %q", col.Name(), "my_int128")
	}
}

func TestUInt128FromBigIntErrors(t *testing.T) {
	if _, err := UInt128FromBigInt(big.NewInt(-1)); err == nil {
		t.Fatal("expected error for negative value")
	}
	maxPlus1 := new(big.Int).Lsh(big.NewInt(1), 128)
	if _, err := UInt128FromBigInt(maxPlus1); err == nil {
		t.Fatal("expected error for value > 2^128-1")
	}
}

func TestInt128FromBigIntErrors(t *testing.T) {
	tooLarge := new(big.Int).Lsh(big.NewInt(1), 127)
	if _, err := Int128FromBigInt(tooLarge); err == nil {
		t.Fatal("expected error for value > 2^127-1")
	}
}
