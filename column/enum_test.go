package column

import (
	"bytes"
	"testing"

	"github.com/ClickHouse/ch-go/proto"
)

func enum8Mapping() *Enum8Column {
	c := NewEnum8Column("v")
	if err := c.Infer(proto.ColumnTypeEnum8.Sub("'a'=1, 'b'=2")); err != nil {
		panic(err)
	}
	return c
}

func enum16Mapping() *Enum16Column {
	c := NewEnum16Column("v")
	if err := c.Infer(proto.ColumnTypeEnum16.Sub("'hello'=100, 'world'=200")); err != nil {
		panic(err)
	}
	return c
}

func TestEnum8RoundTrip(t *testing.T) {
	col := enum8Mapping()
	col.Append("a")
	col.Append("b")
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := enum8Mapping()
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 2); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 2 {
		t.Fatalf("Len: got %d, want 2", got.Len())
	}
	if got.Row(0) != "a" {
		t.Fatalf("Row(0): got %q, want %q", got.Row(0), "a")
	}
	if got.Row(1) != "b" {
		t.Fatalf("Row(1): got %q, want %q", got.Row(1), "b")
	}
}

func TestEnum16RoundTrip(t *testing.T) {
	col := enum16Mapping()
	col.Append("hello")
	col.Append("world")
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := enum16Mapping()
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 2); err != nil {
		t.Fatal(err)
	}
	if got.Row(0) != "hello" {
		t.Fatalf("Row(0): got %q, want %q", got.Row(0), "hello")
	}
	if got.Row(1) != "world" {
		t.Fatalf("Row(1): got %q, want %q", got.Row(1), "world")
	}
}

func TestEnum8AppendUnknownPanic(t *testing.T) {
	col := enum8Mapping()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unknown name")
		}
	}()
	col.Append("unknown")
}

func TestEnum16AppendUnknownPanic(t *testing.T) {
	col := enum16Mapping()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unknown name")
		}
	}()
	col.Append("unknown")
}

func TestEnum8Infer(t *testing.T) {
	col := NewEnum8Column("v")
	if err := col.Infer(proto.ColumnTypeEnum8.Sub("'x'=10, 'y'=20")); err != nil {
		t.Fatal(err)
	}
	if col.Type() != proto.ColumnTypeEnum8.Sub("'x'=10, 'y'=20") {
		t.Fatalf("Type: got %q", col.Type())
	}
	col.Append("x")
	col.Append("y")
	if col.Row(0) != "x" || col.Row(1) != "y" {
		t.Fatalf("Row: got %q, %q", col.Row(0), col.Row(1))
	}
}

func TestEnum16Infer(t *testing.T) {
	col := NewEnum16Column("v")
	if err := col.Infer(proto.ColumnTypeEnum16.Sub("'foo'=1, 'bar'=2")); err != nil {
		t.Fatal(err)
	}
	if col.Type() != proto.ColumnTypeEnum16.Sub("'foo'=1, 'bar'=2") {
		t.Fatalf("Type: got %q", col.Type())
	}
}

func TestEnum8DecodeUnknownRaw(t *testing.T) {
	col := enum8Mapping()
	col.Data = append(col.Data, 99)
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := enum8Mapping()
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 1); err != nil {
		t.Fatal(err)
	}
	if got.Row(0) != "99" {
		t.Fatalf("Row(0): got %q, want %q", got.Row(0), "99")
	}
}

func TestEnum16DecodeUnknownRaw(t *testing.T) {
	col := enum16Mapping()
	col.Data = append(col.Data, 999)
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := enum16Mapping()
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 1); err != nil {
		t.Fatal(err)
	}
	if got.Row(0) != "999" {
		t.Fatalf("Row(0): got %q, want %q", got.Row(0), "999")
	}
}

func TestEnum8TypeBeforeInfer(t *testing.T) {
	col := NewEnum8Column("v")
	if col.Type() != proto.ColumnTypeEnum8 {
		t.Fatalf("Type before Infer: got %q, want %q", col.Type(), proto.ColumnTypeEnum8)
	}
}

func TestEnum16TypeBeforeInfer(t *testing.T) {
	col := NewEnum16Column("v")
	if col.Type() != proto.ColumnTypeEnum16 {
		t.Fatalf("Type before Infer: got %q, want %q", col.Type(), proto.ColumnTypeEnum16)
	}
}

func TestEnum8ZeroRows(t *testing.T) {
	col := enum8Mapping()
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := enum8Mapping()
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 0); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 0 {
		t.Fatalf("Len: got %d, want 0", got.Len())
	}
}

func TestEnum16ZeroRows(t *testing.T) {
	col := enum16Mapping()
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := enum16Mapping()
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 0); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 0 {
		t.Fatalf("Len: got %d, want 0", got.Len())
	}
}

func TestEnum8Nullable(t *testing.T) {
	inner := new(Enum8Column)
	if err := inner.Infer(proto.ColumnTypeEnum8.Sub("'a'=1")); err != nil {
		panic(err)
	}
	col := NewNullable(inner)
	col.Append("a", false)
	col.Append("", true)
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	gotInner := new(Enum8Column)
	if err := gotInner.Infer(proto.ColumnTypeEnum8.Sub("'a'=1")); err != nil {
		panic(err)
	}
	got := NewNullable(gotInner)
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 2); err != nil {
		t.Fatal(err)
	}
	r0, null0 := got.Row(0)
	if null0 {
		t.Fatal("Row(0): expected non-null")
	}
	if r0 != "a" {
		t.Fatalf("Row(0): got %q, want %q", r0, "a")
	}
	r1, null1 := got.Row(1)
	if !null1 {
		t.Fatal("Row(1): expected null")
	}
	// Null entries store raw value 0 (empty string maps to 0), which falls back to "0".
	if r1 != "0" {
		t.Fatalf("Row(1) value: got %q, want %q", r1, "0")
	}
}

func TestEnum16Nullable(t *testing.T) {
	inner := new(Enum16Column)
	if err := inner.Infer(proto.ColumnTypeEnum16.Sub("'x'=1")); err != nil {
		panic(err)
	}
	col := NewNullable(inner)
	col.Append("x", false)
	col.Append("", true)
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	gotInner := new(Enum16Column)
	if err := gotInner.Infer(proto.ColumnTypeEnum16.Sub("'x'=1")); err != nil {
		panic(err)
	}
	got := NewNullable(gotInner)
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 2); err != nil {
		t.Fatal(err)
	}
	r0, null0 := got.Row(0)
	if null0 {
		t.Fatal("Row(0): expected non-null")
	}
	if r0 != "x" {
		t.Fatalf("Row(0): got %q, want %q", r0, "x")
	}
	r1, null1 := got.Row(1)
	if !null1 {
		t.Fatal("Row(1): expected null")
	}
	if r1 != "0" {
		t.Fatalf("Row(1) value: got %q, want %q", r1, "0")
	}
}
