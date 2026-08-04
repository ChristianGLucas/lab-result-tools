package nodes_test

import (
	"context"
	"strings"
	"testing"

	gen "christiangeorgelucas/lab-result-tools/gen"
	"christiangeorgelucas/lab-result-tools/nodes"
)

// TestParseResultValue_Table is the parse table required by the brief:
// every documented value form, unicode micro-sign vs Greek mu, comma
// decimal separators, and whitespace mess.
func TestParseResultValue_Table(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name      string
		text      string
		wantKind  string
		wantValue *float64
		wantLow   *float64
		wantHigh  *float64
		wantQual  string
		wantUnit  string
		wantErr   string // non-empty = expect this error code
	}{
		{name: "plain numeric with unit", text: "5.2 mg/dL", wantKind: "numeric", wantValue: f(5.2), wantUnit: "mg/dL"},
		{name: "bare integer", text: "42", wantKind: "numeric", wantValue: f(42), wantUnit: ""},
		{name: "negative numeric", text: "-3.2", wantKind: "numeric", wantValue: f(-3.2)},
		{name: "comma thousands + decimal", text: "1,234.5", wantKind: "numeric", wantValue: f(1234.5)},
		{name: "censored low", text: "<0.5", wantKind: "censored_low", wantValue: f(0.5)},
		{name: "censored low with unit", text: "<0.5 mg/dL", wantKind: "censored_low", wantValue: f(0.5), wantUnit: "mg/dL"},
		{name: "censored high", text: ">1000", wantKind: "censored_high", wantValue: f(1000)},
		{name: "reported value range", text: "3.4-3.9", wantKind: "range", wantLow: f(3.4), wantHigh: f(3.9)},
		{name: "range with spaces and unit", text: "70 - 110 mg/dL", wantKind: "range", wantLow: f(70), wantHigh: f(110), wantUnit: "mg/dL"},
		{name: "qualitative negative", text: "NEGATIVE", wantKind: "qualitative", wantQual: "NEGATIVE"},
		{name: "qualitative lowercase normalized to upper", text: "negative", wantKind: "qualitative", wantQual: "NEGATIVE"},
		{name: "qualitative trace", text: "trace", wantKind: "qualitative", wantQual: "TRACE"},
		{name: "micro sign unit (U+00B5)", text: "5.2 µg/dL", wantKind: "numeric", wantValue: f(5.2), wantUnit: "μg/dL"},
		{name: "greek mu unit (U+03BC)", text: "5.2 μg/dL", wantKind: "numeric", wantValue: f(5.2), wantUnit: "μg/dL"},
		{name: "surrounding whitespace", text: "  5.2 mg/dL  ", wantKind: "numeric", wantValue: f(5.2), wantUnit: "mg/dL"},
		{name: "internal whitespace mess", text: "5.2\tmg/dL", wantKind: "numeric", wantValue: f(5.2), wantUnit: "mg/dL"},
		{name: "empty is invalid", text: "", wantErr: "INVALID_INPUT"},
		{name: "whitespace only is invalid", text: "   ", wantErr: "INVALID_INPUT"},
		{name: "reversed reported range is invalid", text: "3.9-3.4", wantErr: "RANGE_INVALID"},
		{name: "malformed thousands grouping is invalid, not qualitative", text: "1,23.4", wantErr: "INVALID_INPUT"},
		{name: "european decimal comma is invalid, not qualitative", text: "5,2 mg/dL", wantErr: "INVALID_INPUT"},
		{name: "censored with no number is invalid", text: "<abc", wantErr: "INVALID_INPUT"},
		{name: "all-digit overflow to non-finite is invalid, not qualitative", text: strings.Repeat("9", 400), wantErr: "INVALID_INPUT"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got, err := nodes.ParseResultValue(context.Background(), newTestContext(t), &gen.ParseResultValueRequest{Text: c.text})
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
			if got.GetKind() != c.wantKind {
				t.Fatalf("kind = %q, want %q (full: %+v)", got.GetKind(), c.wantKind, got)
			}
			if c.wantValue != nil {
				if got.Value == nil || *got.Value != *c.wantValue {
					t.Fatalf("value = %v, want %v", got.Value, *c.wantValue)
				}
			}
			if c.wantLow != nil {
				if got.Low == nil || *got.Low != *c.wantLow {
					t.Fatalf("low = %v, want %v", got.Low, *c.wantLow)
				}
			}
			if c.wantHigh != nil {
				if got.High == nil || *got.High != *c.wantHigh {
					t.Fatalf("high = %v, want %v", got.High, *c.wantHigh)
				}
			}
			if c.wantQual != "" && got.GetQualitativeText() != c.wantQual {
				t.Fatalf("qualitative_text = %q, want %q", got.GetQualitativeText(), c.wantQual)
			}
			if got.GetUnit() != c.wantUnit {
				t.Fatalf("unit = %q, want %q", got.GetUnit(), c.wantUnit)
			}
			if got.GetOriginal() != c.text {
				t.Fatalf("original = %q, want %q", got.GetOriginal(), c.text)
			}
		})
	}
}

// TestParseResultValue_NoPanicOnGarbage exercises inputs that a fuzzer or a
// careless upstream system might send, none of which should ever panic.
func TestParseResultValue_NoPanicOnGarbage(t *testing.T) {
	garbage := []string{
		"<", ">", "-", "--", "..", "1-2-3", "1.2.3", ",,,", "<<0.5", ">=<3",
		"NaN", "Infinity", "-Infinity", "1e400", "🧪🧪🧪", " ", "<>",
	}
	for _, g := range garbage {
		g := g
		t.Run(g, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked on %q: %v", g, r)
				}
			}()
			got, err := nodes.ParseResultValue(context.Background(), newTestContext(t), &gen.ParseResultValueRequest{Text: g})
			if err != nil {
				t.Fatalf("unexpected transport error on %q: %v", g, err)
			}
			if got == nil {
				t.Fatalf("nil result on %q", g)
			}
		})
	}
}
