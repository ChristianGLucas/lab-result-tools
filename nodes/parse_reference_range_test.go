package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/lab-result-tools/gen"
	"christiangeorgelucas/lab-result-tools/nodes"
)

func TestParseReferenceRange_Table(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name         string
		text         string
		wantLow      *float64
		wantHigh     *float64
		wantLowIncl  bool
		wantHighIncl bool
		wantUnit     string
		wantErr      string
	}{
		{name: "bare range both inclusive", text: "3.5-5.0", wantLow: f(3.5), wantHigh: f(5.0), wantLowIncl: true, wantHighIncl: true},
		{name: "bare range with unit and spaces", text: "70 - 110 mg/dL", wantLow: f(70), wantHigh: f(110), wantLowIncl: true, wantHighIncl: true, wantUnit: "mg/dL"},
		{name: "less-than is high exclusive", text: "<200", wantHigh: f(200), wantHighIncl: false},
		{name: "less-equal is high inclusive", text: "<=200", wantHigh: f(200), wantHighIncl: true},
		{name: "greater-than is low exclusive", text: ">40", wantLow: f(40), wantLowIncl: false},
		{name: "greater-equal is low inclusive", text: ">=40", wantLow: f(40), wantLowIncl: true},
		{name: "micro sign unit normalized", text: "3.5-5.0 µg/L", wantLow: f(3.5), wantHigh: f(5.0), wantLowIncl: true, wantHighIncl: true, wantUnit: "μg/L"},
		{name: "empty is invalid", text: "", wantErr: "INVALID_INPUT"},
		{name: "reversed range is invalid", text: "5.0-3.5", wantErr: "RANGE_INVALID"},
		{name: "garbage is invalid", text: "banana", wantErr: "INVALID_INPUT"},
		{name: "operator with no number is invalid", text: "<=", wantErr: "INVALID_INPUT"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got, err := nodes.ParseReferenceRange(context.Background(), newTestContext(t), &gen.ParseReferenceRangeRequest{Text: c.text})
			if err != nil {
				t.Fatalf("unexpected transport error: %v", err)
			}
			if c.wantErr != "" {
				if got.GetError() == nil || got.GetError().GetCode() != c.wantErr {
					t.Fatalf("want error %q, got %+v", c.wantErr, got)
				}
				return
			}
			if got.GetError() != nil {
				t.Fatalf("unexpected error: %+v", got.GetError())
			}
			if c.wantLow == nil && got.Low != nil {
				t.Fatalf("low = %v, want absent", *got.Low)
			}
			if c.wantLow != nil && (got.Low == nil || *got.Low != *c.wantLow) {
				t.Fatalf("low = %v, want %v", got.Low, *c.wantLow)
			}
			if c.wantHigh == nil && got.High != nil {
				t.Fatalf("high = %v, want absent", *got.High)
			}
			if c.wantHigh != nil && (got.High == nil || *got.High != *c.wantHigh) {
				t.Fatalf("high = %v, want %v", got.High, *c.wantHigh)
			}
			if got.GetLowInclusive() != c.wantLowIncl {
				t.Fatalf("low_inclusive = %v, want %v", got.GetLowInclusive(), c.wantLowIncl)
			}
			if got.GetHighInclusive() != c.wantHighIncl {
				t.Fatalf("high_inclusive = %v, want %v", got.GetHighInclusive(), c.wantHighIncl)
			}
			if got.GetUnit() != c.wantUnit {
				t.Fatalf("unit = %q, want %q", got.GetUnit(), c.wantUnit)
			}
		})
	}
}

// TestParseReferenceRange_NoPanicOnGarbage mirrors the ParseResultValue
// no-panic sweep for the range grammar's operator set.
func TestParseReferenceRange_NoPanicOnGarbage(t *testing.T) {
	garbage := []string{"<", ">", "<=", ">=", "-", "--", "1-2-3", ",,,", "🧪", " ", "<<5"}
	for _, g := range garbage {
		g := g
		t.Run(g, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked on %q: %v", g, r)
				}
			}()
			got, err := nodes.ParseReferenceRange(context.Background(), newTestContext(t), &gen.ParseReferenceRangeRequest{Text: g})
			if err != nil {
				t.Fatalf("unexpected transport error on %q: %v", g, err)
			}
			if got == nil {
				t.Fatalf("nil result on %q", g)
			}
		})
	}
}
