package column

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/shopspring/decimal"
)

type decimalCase struct {
	name   string
	newCol func() Column
	ct     proto.ColumnType
	prec   int
	scale  int
}

var decimalTypes = []decimalCase{
	{"Decimal32", func() Column { return NewDecimal32Column("") }, proto.ColumnTypeDecimal32, 9, 2},
	{"Decimal64", func() Column { return NewDecimal64Column("") }, proto.ColumnTypeDecimal64, 18, 4},
	{"Decimal128", func() Column { return NewDecimal128Column("") }, proto.ColumnTypeDecimal128, 38, 10},
	{"Decimal256", func() Column { return NewDecimal256Column("") }, proto.ColumnTypeDecimal256, 76, 20},
}

func TestDecimalRoundTrip(t *testing.T) {
	for _, tc := range decimalTypes {
		t.Run(tc.name+"/", func(t *testing.T) {
			col := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Append(decimal.Decimal)
				Row(int) decimal.Decimal
				EncodeColumn(*proto.Buffer) error
				DecodeColumn(*proto.Reader, int) error
				Reset()
				Len() int
			})
			ct := tc.ct.With(
				fmt.Sprintf("%d", tc.prec),
				fmt.Sprintf("%d", tc.scale),
			)
			if err := col.Infer(ct); err != nil {
				t.Fatal(err)
			}
			input := decimal.NewFromFloat(12.34) // exact at any scale
			col.Append(input)
			var buf proto.Buffer
			if err := col.EncodeColumn(&buf); err != nil {
				t.Fatal(err)
			}
			got := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Row(int) decimal.Decimal
				DecodeColumn(*proto.Reader, int) error
				Len() int
			})
			if err := got.Infer(ct); err != nil {
				t.Fatal(err)
			}
			if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), col.Len()); err != nil {
				t.Fatal(err)
			}
			want := input.Round(int32(tc.scale))
			if got.Row(0).Cmp(want) != 0 {
				t.Fatalf("Row(0): got %s, want %s", got.Row(0).String(), want.String())
			}
		})
	}
}

func TestDecimalInferPrecisionPanic(t *testing.T) {
	cases := []struct {
		name string
		col  func() interface{ Infer(proto.ColumnType) error }
		ct   proto.ColumnType
		bad  int
	}{
		{"Decimal32", func() interface{ Infer(proto.ColumnType) error } { return NewDecimal32Column("") }, proto.ColumnTypeDecimal32, 10},
		{"Decimal64", func() interface{ Infer(proto.ColumnType) error } { return NewDecimal64Column("") }, proto.ColumnTypeDecimal64, 19},
		{"Decimal128", func() interface{ Infer(proto.ColumnType) error } { return NewDecimal128Column("") }, proto.ColumnTypeDecimal128, 39},
		{"Decimal256", func() interface{ Infer(proto.ColumnType) error } { return NewDecimal256Column("") }, proto.ColumnTypeDecimal256, 77},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expected panic")
				}
			}()
			ct := tc.ct.With(fmt.Sprintf("%d", tc.bad), "0")
			tc.col().Infer(ct)
		})
	}
}

func TestDecimalType(t *testing.T) {
	for _, tc := range decimalTypes {
		t.Run(tc.name, func(t *testing.T) {
			col := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Type() proto.ColumnType
			})
			ct := tc.ct.With(
				fmt.Sprintf("%d", tc.prec),
				fmt.Sprintf("%d", tc.scale),
			)
			if err := col.Infer(ct); err != nil {
				t.Fatal(err)
			}
			got := col.Type()
			if got != ct {
				t.Fatalf("Type(): got %q, want %q", got, ct)
			}
		})
	}
}

func TestDecimalZeroRows(t *testing.T) {
	for _, tc := range decimalTypes {
		t.Run(tc.name, func(t *testing.T) {
			col := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				EncodeColumn(*proto.Buffer) error
				DecodeColumn(*proto.Reader, int) error
				Len() int
			})
			ct := tc.ct.With(
				fmt.Sprintf("%d", tc.prec),
				fmt.Sprintf("%d", tc.scale),
			)
			if err := col.Infer(ct); err != nil {
				t.Fatal(err)
			}
			var buf proto.Buffer
			if err := col.EncodeColumn(&buf); err != nil {
				t.Fatal(err)
			}
			got := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				DecodeColumn(*proto.Reader, int) error
				Len() int
			})
			if err := got.Infer(ct); err != nil {
				t.Fatal(err)
			}
			if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 0); err != nil {
				t.Fatal(err)
			}
			if got.Len() != 0 {
				t.Fatalf("Len(): got %d, want 0", got.Len())
			}
		})
	}
}

func TestDecimalAppendScale(t *testing.T) {
	col := NewDecimal64Column("")
	if err := col.Infer(proto.ColumnTypeDecimal64.With("18", "6")); err != nil {
		t.Fatal(err)
	}
	col.Append(decimal.NewFromFloat(1.5))
	if len(col.Data) != 1 {
		t.Fatalf("len(Data): got %d, want 1", len(col.Data))
	}
	if col.Data[0] != 1500000 {
		t.Fatalf("backing int64: got %d, want 1500000", col.Data[0])
	}
}

type inferOf[T any] interface {
	Of[T]
	Infer(proto.ColumnType) error
}

func TestDecimalNullable(t *testing.T) {
	for _, tc := range decimalTypes {
		t.Run(tc.name, func(t *testing.T) {
			inner := tc.newCol().(inferOf[decimal.Decimal])
			ct := tc.ct.With(
				fmt.Sprintf("%d", tc.prec),
				fmt.Sprintf("%d", tc.scale),
			)
			if err := inner.Infer(ct); err != nil {
				t.Fatal(err)
			}
			col := NewNullable[decimal.Decimal](inner)
			val := decimal.NewFromFloat(42.5)
			col.Append(val, false)
			var zero decimal.Decimal
			col.Append(zero, true)
			var buf proto.Buffer
			if err := col.EncodeColumn(&buf); err != nil {
				t.Fatal(err)
			}
			gotInner := tc.newCol().(inferOf[decimal.Decimal])
			if err := gotInner.Infer(ct); err != nil {
				t.Fatal(err)
			}
			got := NewNullable[decimal.Decimal](gotInner)
			if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 2); err != nil {
				t.Fatal(err)
			}
			r0, null0 := got.Row(0)
			if null0 {
				t.Fatal("Row(0): expected non-null")
			}
			if r0.Cmp(val.Round(int32(tc.scale))) != 0 {
				t.Fatalf("Row(0): got %s, want %s", r0.String(), val.String())
			}
			_, null1 := got.Row(1)
			if !null1 {
				t.Fatal("Row(1): expected null")
			}
		})
	}
}

func TestDecimalWireMatch(t *testing.T) {
	t.Run("Decimal32_vs_Int32", func(t *testing.T) {
		d := NewDecimal32Column("")
		if err := d.Infer(proto.ColumnTypeDecimal32.With("9", "2")); err != nil {
			t.Fatal(err)
		}
		d.Append(decimal.NewFromFloat(42))
		var dBuf proto.Buffer
		if err := d.EncodeColumn(&dBuf); err != nil {
			t.Fatal(err)
		}
		i := NewBaseColumn[int32]("")
		i.Append(4200)
		var iBuf proto.Buffer
		if err := i.EncodeColumn(&iBuf); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(dBuf.Buf, iBuf.Buf) {
			t.Fatal("Decimal32 and Int32 wire bytes differ")
		}
	})
	t.Run("Decimal64_vs_Int64", func(t *testing.T) {
		d := NewDecimal64Column("")
		if err := d.Infer(proto.ColumnTypeDecimal64.With("18", "0")); err != nil {
			t.Fatal(err)
		}
		d.Append(decimal.NewFromFloat(999999))
		var dBuf proto.Buffer
		if err := d.EncodeColumn(&dBuf); err != nil {
			t.Fatal(err)
		}
		i := NewBaseColumn[int64]("")
		i.Append(999999)
		var iBuf proto.Buffer
		if err := i.EncodeColumn(&iBuf); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(dBuf.Buf, iBuf.Buf) {
			t.Fatal("Decimal64 and Int64 wire bytes differ")
		}
	})
}

func TestDecimalInferGenericDecimal(t *testing.T) {
	for _, tc := range decimalTypes {
		t.Run(tc.name, func(t *testing.T) {
			col := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Type() proto.ColumnType
			})
			ct := tc.ct.With(
				fmt.Sprintf("%d", tc.prec),
				fmt.Sprintf("%d", tc.scale),
			)
			if err := col.Infer(proto.ColumnTypeDecimal.With(
				fmt.Sprintf("%d", tc.prec),
				fmt.Sprintf("%d", tc.scale),
			)); err != nil {
				t.Fatal(err)
			}
			got := col.Type()
			if got != ct {
				t.Fatalf("Type(): got %q, want %q (after generic Decimal infer)", got, ct)
			}
		})
	}
}

func TestDecimalScaleZero(t *testing.T) {
	for _, tc := range decimalTypes {
		if tc.prec > 18 {
			continue // Decimal128/256 need big.Int
		}
		t.Run(tc.name+"/", func(t *testing.T) {
			col := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Append(decimal.Decimal)
				Row(int) decimal.Decimal
				EncodeColumn(*proto.Buffer) error
				DecodeColumn(*proto.Reader, int) error
				Len() int
			})
			ct := tc.ct.With(
				fmt.Sprintf("%d", tc.prec),
				"0",
			)
			if err := col.Infer(ct); err != nil {
				t.Fatal(err)
			}
			col.Append(decimal.NewFromFloat(42))
			var buf proto.Buffer
			if err := col.EncodeColumn(&buf); err != nil {
				t.Fatal(err)
			}
			got := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Row(int) decimal.Decimal
				DecodeColumn(*proto.Reader, int) error
				Len() int
			})
			if err := got.Infer(ct); err != nil {
				t.Fatal(err)
			}
			if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), col.Len()); err != nil {
				t.Fatal(err)
			}
			if got.Row(0).Cmp(decimal.New(42, 0)) != 0 {
				t.Fatalf("Row(0): got %s, want 42", got.Row(0).String())
			}
		})
	}
}

func TestDecimalInferNoParams(t *testing.T) {
	for _, tc := range decimalTypes {
		t.Run(tc.name, func(t *testing.T) {
			col := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Append(decimal.Decimal)
				Row(int) decimal.Decimal
				EncodeColumn(*proto.Buffer) error
				DecodeColumn(*proto.Reader, int) error
				Len() int
			})
			if err := col.Infer(tc.ct); err != nil {
				t.Fatal(err)
			}
			col.Append(decimal.New(0, 0))
			var buf proto.Buffer
			if err := col.EncodeColumn(&buf); err != nil {
				t.Fatal(err)
			}
			got := tc.newCol().(interface {
				Infer(proto.ColumnType) error
				Row(int) decimal.Decimal
				DecodeColumn(*proto.Reader, int) error
				Len() int
			})
			if err := got.Infer(tc.ct); err != nil {
				t.Fatal(err)
			}
			if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), col.Len()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDecimalNegative(t *testing.T) {
	col := NewDecimal64Column("")
	if err := col.Infer(proto.ColumnTypeDecimal64.With("18", "2")); err != nil {
		t.Fatal(err)
	}
	col.Append(decimal.NewFromFloat(-10.5))
	var buf proto.Buffer
	if err := col.EncodeColumn(&buf); err != nil {
		t.Fatal(err)
	}
	got := NewDecimal64Column("")
	if err := got.Infer(proto.ColumnTypeDecimal64.With("18", "2")); err != nil {
		t.Fatal(err)
	}
	if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), col.Len()); err != nil {
		t.Fatal(err)
	}
	if got.Row(0).Cmp(decimal.NewFromFloat(-10.50)) != 0 {
		t.Fatalf("Row(0): got %s, want -10.50", got.Row(0).String())
	}
}

func TestDecimalEdgeCases(t *testing.T) {
	t.Run("max_int32_decimal32", func(t *testing.T) {
		col := NewDecimal32Column("")
		if err := col.Infer(proto.ColumnTypeDecimal32.With("9", "0")); err != nil {
			t.Fatal(err)
		}
		col.Append(decimal.New(math.MaxInt32, 0))
		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}
		got := NewDecimal32Column("")
		if err := got.Infer(proto.ColumnTypeDecimal32.With("9", "0")); err != nil {
			t.Fatal(err)
		}
		if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 1); err != nil {
			t.Fatal(err)
		}
		if got.Row(0).Cmp(decimal.New(math.MaxInt32, 0)) != 0 {
			t.Fatalf("Row(0): got %s, want %d", got.Row(0).String(), math.MaxInt32)
		}
	})

	t.Run("large_scale_round_trip", func(t *testing.T) {
		col := NewDecimal64Column("")
		if err := col.Infer(proto.ColumnTypeDecimal64.With("18", "9")); err != nil {
			t.Fatal(err)
		}
		col.Append(decimal.NewFromFloat(1.23456789))
		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}
		got := NewDecimal64Column("")
		if err := got.Infer(proto.ColumnTypeDecimal64.With("18", "9")); err != nil {
			t.Fatal(err)
		}
		if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 1); err != nil {
			t.Fatal(err)
		}
		want := decimal.RequireFromString("1.23456789")
		if got.Row(0).Cmp(want) != 0 {
			t.Fatalf("Row(0): got %s, want %s", got.Row(0).String(), want.String())
		}
	})

	t.Run("decimal128_zero", func(t *testing.T) {
		col := NewDecimal128Column("")
		if err := col.Infer(proto.ColumnTypeDecimal128.With("38", "10")); err != nil {
			t.Fatal(err)
		}
		col.Append(decimal.New(0, 0))
		var buf proto.Buffer
		if err := col.EncodeColumn(&buf); err != nil {
			t.Fatal(err)
		}
		got := NewDecimal128Column("")
		if err := got.Infer(proto.ColumnTypeDecimal128.With("38", "10")); err != nil {
			t.Fatal(err)
		}
		if err := got.DecodeColumn(proto.NewReader(bytes.NewReader(buf.Buf)), 1); err != nil {
			t.Fatal(err)
		}
		if !got.Row(0).IsZero() {
			t.Fatalf("Row(0): expected zero, got %s", got.Row(0).String())
		}
	})
}
