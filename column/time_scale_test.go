package column

import (
	"bytes"
	"testing"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

func mustTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func TestDate32RoundTrip(t *testing.T) {
	dates := []time.Time{
		mustTime("1970-01-01 00:00:00"),
		mustTime("2024-03-14 00:00:00"),
		mustTime("1969-12-31 00:00:00"),
		mustTime("2000-01-01 00:00:00"),
	}
	col := NewDate32Column("test")
	for _, d := range dates {
		col.Append(d)
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewDate32Column("test")
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(dates)); err != nil {
		t.Fatal(err)
	}
	if got.Len() != len(dates) {
		t.Fatalf("Len: got %d, want %d", got.Len(), len(dates))
	}
	for i, d := range dates {
		r := got.Row(i)
		if r.Year() != d.Year() || r.YearDay() != d.YearDay() {
			t.Fatalf("Row(%d): got %v, want %v (date)", i, r, d)
		}
	}
}

func TestDate32ZeroRows(t *testing.T) {
	col := NewDate32Column("test")
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewDate32Column("test")
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 0); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 0 {
		t.Fatalf("Len: got %d, want 0", got.Len())
	}
}

func TestDate32Nullable(t *testing.T) {
	inner := NewDate32Column("v")
	col := NewNullable(inner)
	dates := []time.Time{
		mustTime("2024-01-01 00:00:00"),
		mustTime("2024-06-15 00:00:00"),
	}
	nulls := []bool{false, true}
	for i, d := range dates {
		col.Append(d, nulls[i])
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewNullable(NewDate32Column("v"))
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(dates)); err != nil {
		t.Fatal(err)
	}
	for i, d := range dates {
		r, isNull := got.Row(i)
		if isNull != nulls[i] {
			t.Fatalf("Row(%d) null: got %v, want %v", i, isNull, nulls[i])
		}
		if !isNull && (r.Year() != d.Year() || r.YearDay() != d.YearDay()) {
			t.Fatalf("Row(%d): got %v, want %v (date)", i, r, d)
		}
	}
}

func TestDateTime64RoundTripScale3(t *testing.T) {
	times := []time.Time{
		time.Unix(1700000000, 123000000).UTC(),
		time.Unix(1700000001, 456000000).UTC(),
		time.Unix(0, 0).UTC(),
	}
	col := NewDateTime64Column("test", 3)
	for _, v := range times {
		col.Append(v)
	}
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewDateTime64Column("test", 3)
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), len(times)); err != nil {
		t.Fatal(err)
	}
	for i, v := range times {
		r := got.Row(i)
		if r.UnixMilli() != v.UnixMilli() {
			t.Fatalf("Row(%d): got %v (ms %d), want %v (ms %d)", i, r, r.UnixMilli(), v, v.UnixMilli())
		}
	}
}

func TestDateTime64VariousScales(t *testing.T) {
	scales := []proto.Precision{0, 3, 6, 9}
	base := time.Unix(1700000000, 987654321).UTC()

	for _, scale := range scales {
		col := NewDateTime64Column("test", scale)
		col.Append(base)
		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatalf("scale %d encode: %v", scale, err)
		}
		got := NewDateTime64Column("test", scale)
		if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 1); err != nil {
			t.Fatalf("scale %d decode: %v", scale, err)
		}
		r := got.Row(0)
		gotNano := r.UnixNano()
		wantNano := base.UnixNano() - base.UnixNano()%scale.Duration().Nanoseconds()
		if gotNano != wantNano {
			t.Errorf("scale %d: got %d ns, want %d ns (truncated)", scale, gotNano, wantNano)
		}
	}
}

func TestDateTime64ZeroRows(t *testing.T) {
	col := NewDateTime64Column("test", 3)
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewDateTime64Column("test", 3)
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 0); err != nil {
		t.Fatal(err)
	}
	if got.Len() != 0 {
		t.Fatalf("Len: got %d, want 0", got.Len())
	}
}

func TestDateTime64Infer(t *testing.T) {
	col := NewDateTime64Column("test", 0)
	if err := col.Infer(proto.ColumnTypeDateTime64.Sub("3")); err != nil {
		t.Fatal(err)
	}
	if col.Precision != 3 {
		t.Fatalf("Precision: got %d, want 3", col.Precision)
	}
	if col.Location != nil {
		t.Fatalf("Location: got %v, want nil", col.Location)
	}
}

func TestDateTime64InferWithLocation(t *testing.T) {
	col := NewDateTime64Column("test", 0)
	if err := col.Infer(proto.ColumnTypeDateTime64.Sub("3", "'UTC'")); err != nil {
		t.Fatal(err)
	}
	if col.Precision != 3 {
		t.Fatalf("Precision: got %d, want 3", col.Precision)
	}
	if col.Location == nil || col.Location.String() != "UTC" {
		t.Fatalf("Location: got %v, want UTC", col.Location)
	}
}

func TestDateTime64TypeString(t *testing.T) {
	col := NewDateTime64Column("test", 6)
	if col.Type() != proto.ColumnTypeDateTime64.Sub("6") {
		t.Fatalf("Type: got %q, want %q", col.Type(), proto.ColumnTypeDateTime64.Sub("6"))
	}
}

func TestDateTime64TypeStringWithLocation(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	col := NewDateTime64Column("test", 3)
	col.Location = loc
	got := string(col.Type())
	want := "DateTime64(3, 'America/New_York')"
	if got != want {
		t.Fatalf("Type: got %q, want %q", got, want)
	}
}
