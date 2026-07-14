package column

import (
	"bytes"
	"math"
	"testing"

	"github.com/ClickHouse/ch-go/proto"
)

func roundTrip[T comparable](t *testing.T, col Of[T], vals []T) {
	t.Helper()
	for _, v := range vals {
		col.Append(v)
	}

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	dst := NewBaseColumn[T]("test")
	if err := dst.DecodeColumn(r, len(vals)); err != nil {
		t.Fatal(err)
	}
	if dst.Len() != len(vals) {
		t.Fatalf("len: got %d, want %d", dst.Len(), len(vals))
	}
	for i, expected := range vals {
		if dst.Row(i) != expected {
			t.Fatalf("row %d: got %v, want %v", i, dst.Row(i), expected)
		}
	}
}

func TestBaseRoundTrip(t *testing.T) {
	t.Run("uint8", func(t *testing.T) { roundTrip(t, NewBaseColumn[uint8]("v"), []uint8{1, 2, 255}) })
	t.Run("uint16", func(t *testing.T) { roundTrip(t, NewBaseColumn[uint16]("v"), []uint16{1, 256, 65535}) })
	t.Run("uint32", func(t *testing.T) { roundTrip(t, NewBaseColumn[uint32]("v"), []uint32{1, 70000, math.MaxUint32}) })
	t.Run("uint64", func(t *testing.T) { roundTrip(t, NewBaseColumn[uint64]("v"), []uint64{1, math.MaxUint64}) })
	t.Run("int8", func(t *testing.T) { roundTrip(t, NewBaseColumn[int8]("v"), []int8{-128, 0, 127}) })
	t.Run("int16", func(t *testing.T) { roundTrip(t, NewBaseColumn[int16]("v"), []int16{-32768, 0, 32767}) })
	t.Run("int32", func(t *testing.T) { roundTrip(t, NewBaseColumn[int32]("v"), []int32{math.MinInt32, 0, math.MaxInt32}) })
	t.Run("int64", func(t *testing.T) { roundTrip(t, NewBaseColumn[int64]("v"), []int64{math.MinInt64, 0, math.MaxInt64}) })
	t.Run("float32", func(t *testing.T) {
		roundTrip(t, NewBaseColumn[float32]("v"), []float32{0, 3.14, -2.5, math.MaxFloat32})
	})
	t.Run("float64", func(t *testing.T) {
		roundTrip(t, NewBaseColumn[float64]("v"), []float64{0, 3.14159265359, -2.5, math.MaxFloat64})
	})
}

func TestBaseData(t *testing.T) {
	col := NewBaseColumn[uint64]("id")
	col.Append(10)
	col.Append(20)
	col.Append(30)

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewBaseColumn[uint64]("id")
	if err := got.DecodeColumn(r, 3); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 3 {
		t.Fatalf("len: got %d, want 3", got.Len())
	}

	du := got.Data
	if len(du) != 3 {
		t.Fatalf("Data len: got %d, want 3", len(du))
	}
	if du[0] != got.Row(0) || du[1] != got.Row(1) || du[2] != got.Row(2) {
		t.Fatal("Data values differ from Row")
	}
}

func TestStrRoundTrip(t *testing.T) {
	col := NewStrColumn("s")
	col.Append("hello")
	col.Append("")
	col.Append("world")

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewStrColumn("s")
	if err := got.DecodeColumn(r, 3); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 3 {
		t.Fatalf("len: got %d, want 3", got.Len())
	}

	cases := []struct {
		i    int
		want string
	}{
		{0, "hello"},
		{1, ""},
		{2, "world"},
	}
	for _, c := range cases {
		if got.Row(c.i) != c.want {
			t.Fatalf("row %d: got %q, want %q", c.i, got.Row(c.i), c.want)
		}
	}
}

func TestNullableRoundTrip(t *testing.T) {
	col := NewNullable(NewBaseColumn[uint64]("v"))
	col.Append(1, false)
	col.Append(0, true)
	col.Append(3, false)

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewNullable(NewBaseColumn[uint64]("v"))
	if err := got.DecodeColumn(r, 3); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 3 {
		t.Fatalf("len: got %d, want 3", got.Len())
	}

	v0, n0 := got.Row(0)
	if v0 != 1 || n0 {
		t.Fatalf("row 0: got (%d, %v), want (1, false)", v0, n0)
	}
	v1, n1 := got.Row(1)
	if v1 != 0 || !n1 {
		t.Fatalf("row 1: got (%d, %v), want (0, true)", v1, n1)
	}
	v2, n2 := got.Row(2)
	if v2 != 3 || n2 {
		t.Fatalf("row 2: got (%d, %v), want (3, false)", v2, n2)
	}
}

func TestLowCardinalityRoundTrip(t *testing.T) {
	t.Run("uint8", func(t *testing.T) {
		base := NewBaseColumn[uint8]("v")
		col := NewLowCardinality(base)
		base.AppendArr([]uint8{1, 2, 3, 1, 2, 3})

		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}

		r := proto.NewReader(bytes.NewReader(buf.Buf))
		got := NewLowCardinality(NewBaseColumn[uint8]("v"))
		if err := got.DecodeColumn(r, 6); err != nil {
			t.Fatal(err)
		}
		if got.Len() != 6 {
			t.Fatalf("len: got %d, want 6", got.Len())
		}
		for i, want := range []uint8{1, 2, 3, 1, 2, 3} {
			if got.Row(i) != want {
				t.Fatalf("row %d: got %d, want %d", i, got.Row(i), want)
			}
		}
	})

	t.Run("string", func(t *testing.T) {
		s := NewStrColumn("v")
		col := NewLowCardinality(s)
		s.AppendArr([]string{"a", "b", "a", "c"})

		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}

		r := proto.NewReader(bytes.NewReader(buf.Buf))
		got := NewLowCardinality(NewStrColumn("v"))
		if err := got.DecodeColumn(r, 4); err != nil {
			t.Fatal(err)
		}
		if got.Len() != 4 {
			t.Fatalf("len: got %d, want 4", got.Len())
		}
		for i, want := range []string{"a", "b", "a", "c"} {
			if got.Row(i) != want {
				t.Fatalf("row %d: got %q, want %q", i, got.Row(i), want)
			}
		}
	})
}

func TestTupleRoundTrip(t *testing.T) {
	col := NewTuple2(NewBaseColumn[uint64]("a"), NewStrColumn("b"))
	col.Append(Tuple2Value[uint64, string]{1, "one"})
	col.Append(Tuple2Value[uint64, string]{2, "two"})

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	got := NewTuple2(NewBaseColumn[uint64]("a"), NewStrColumn("b"))
	r := proto.NewReader(bytes.NewReader(buf.Buf))
	if err := got.DecodeColumn(r, 2); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 2 {
		t.Fatalf("len: got %d, want 2", got.Len())
	}

	r0 := got.Row(0)
	if r0.T1 != 1 || r0.T2 != "one" {
		t.Fatalf("row 0: got (%d, %q), want (1, one)", r0.T1, r0.T2)
	}
	r1 := got.Row(1)
	if r1.T1 != 2 || r1.T2 != "two" {
		t.Fatalf("row 1: got (%d, %q), want (2, two)", r1.T1, r1.T2)
	}
}

func TestEmptyColumn(t *testing.T) {
	t.Run("Base[uint64]", func(t *testing.T) {
		col := NewBaseColumn[uint64]("v")
		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}
		r := proto.NewReader(bytes.NewReader(buf.Buf))
		got := NewBaseColumn[uint64]("v")
		if err := got.DecodeColumn(r, 0); err != nil {
			t.Fatal(err)
		}
		if got.Len() != 0 {
			t.Fatalf("len: got %d, want 0", got.Len())
		}
	})

	t.Run("Str", func(t *testing.T) {
		col := NewStrColumn("v")
		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}
		r := proto.NewReader(bytes.NewReader(buf.Buf))
		got := NewStrColumn("v")
		if err := got.DecodeColumn(r, 0); err != nil {
			t.Fatal(err)
		}
		if got.Len() != 0 {
			t.Fatalf("len: got %d, want 0", got.Len())
		}
	})

	t.Run("Nullable[uint64]", func(t *testing.T) {
		col := NewNullable(NewBaseColumn[uint64]("v"))
		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}
		r := proto.NewReader(bytes.NewReader(buf.Buf))
		got := NewNullable(NewBaseColumn[uint64]("v"))
		if err := got.DecodeColumn(r, 0); err != nil {
			t.Fatal(err)
		}
		if got.Len() != 0 {
			t.Fatalf("len: got %d, want 0", got.Len())
		}
	})
}

func TestBaseType(t *testing.T) {
	tests := []struct {
		col  Column
		want proto.ColumnType
	}{
		{NewBaseColumn[uint8]("v"), proto.ColumnTypeUInt8},
		{NewBaseColumn[uint16]("v"), proto.ColumnTypeUInt16},
		{NewBaseColumn[uint32]("v"), proto.ColumnTypeUInt32},
		{NewBaseColumn[uint64]("v"), proto.ColumnTypeUInt64},
		{NewBaseColumn[int8]("v"), proto.ColumnTypeInt8},
		{NewBaseColumn[int16]("v"), proto.ColumnTypeInt16},
		{NewBaseColumn[int32]("v"), proto.ColumnTypeInt32},
		{NewBaseColumn[int64]("v"), proto.ColumnTypeInt64},
		{NewBaseColumn[float32]("v"), proto.ColumnTypeFloat32},
		{NewBaseColumn[float64]("v"), proto.ColumnTypeFloat64},
		{NewBaseColumn[string]("v"), proto.ColumnType("")},
	}
	for _, tt := range tests {
		if got := tt.col.Type(); got != tt.want {
			t.Fatalf("Type() = %q, want %q", got, tt.want)
		}
	}
}

func TestBaseUnsupportedType(t *testing.T) {
	col := NewBaseColumn[string]("v")
	if got := col.Type(); got != "" {
		t.Fatalf("unexpected type for string: %q", got)
	}
}

func TestName(t *testing.T) {
	if got := NewBaseColumn[uint64]("id").Name(); got != "id" {
		t.Fatalf("Name: got %q, want id", got)
	}
	if got := NewStrColumn("s").Name(); got != "s" {
		t.Fatalf("Name: got %q, want s", got)
	}
	if got := NewNullable(NewBaseColumn[uint64]("v")).Name(); got != "v" {
		t.Fatalf("Name: got %q, want v", got)
	}
	if got := NewLowCardinality(NewBaseColumn[uint64]("v")).Name(); got != "v" {
		t.Fatalf("Name: got %q, want v", got)
	}
	tup := NewTuple2(NewBaseColumn[uint64]("a"), NewStrColumn("b"))
	if got := tup.Name(); got != "" {
		t.Fatalf("Tuple Name: got %q, want empty", got)
	}
}

func TestAppendArr(t *testing.T) {
	col := NewBaseColumn[uint64]("v")
	col.AppendArr([]uint64{1, 2, 3})
	if col.Len() != 3 {
		t.Fatalf("len: got %d, want 3", col.Len())
	}
	if col.Row(0) != 1 || col.Row(2) != 3 {
		t.Fatal("values mismatch after AppendArr")
	}
}

func TestSingleton(t *testing.T) {
	col := NewBaseColumn[uint64]("v")
	col.Append(42)

	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}

	r := proto.NewReader(bytes.NewReader(buf.Buf))
	got := NewBaseColumn[uint64]("v")
	if err := got.DecodeColumn(r, 1); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 1 {
		t.Fatalf("len: got %d, want 1", got.Len())
	}
	if got.Row(0) != 42 {
		t.Fatalf("row 0: got %d, want 42", got.Row(0))
	}
}
