package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/lab-result-tools/gen"
	"christiangeorgelucas/lab-result-tools/nodes"
)

func f64(v float64) *float64 { return &v }

func numeric(v float64, unit string) *gen.ParsedValue {
	return &gen.ParsedValue{Kind: "numeric", Value: f64(v), Unit: unit}
}
func censoredLow(t float64, unit string) *gen.ParsedValue {
	return &gen.ParsedValue{Kind: "censored_low", Value: f64(t), Unit: unit}
}
func censoredHigh(t float64, unit string) *gen.ParsedValue {
	return &gen.ParsedValue{Kind: "censored_high", Value: f64(t), Unit: unit}
}
func rangeValue(lo, hi float64, unit string) *gen.ParsedValue {
	return &gen.ParsedValue{Kind: "range", Low: f64(lo), High: f64(hi), Unit: unit}
}
func qualitative(text string) *gen.ParsedValue {
	return &gen.ParsedValue{Kind: "qualitative", QualitativeText: text}
}

// rng builds a ParsedRange; pass nil for an absent bound.
func rng(low, high *float64, lowIncl, highIncl bool, unit string) *gen.ParsedRange {
	return &gen.ParsedRange{Low: low, High: high, LowInclusive: lowIncl, HighInclusive: highIncl, Unit: unit}
}

func evaluate(t *testing.T, value *gen.ParsedValue, r *gen.ParsedRange, critical *gen.ParsedRange) *gen.EvaluationResult {
	t.Helper()
	got, err := nodes.EvaluateResult(context.Background(), newTestContext(t), &gen.EvaluateResultRequest{
		Value: value, Range: r, CriticalRange: critical,
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	return got
}

// TestEvaluateResult_BoundaryMatrix exercises exactly-at-bound, just-inside,
// and just-outside for both inclusive and exclusive boundaries — the
// classic fencepost bug the brief calls out.
func TestEvaluateResult_BoundaryMatrix(t *testing.T) {
	inclusiveRange := rng(f64(3.5), f64(5.0), true, true, "")

	cases := []struct {
		name      string
		value     float64
		wantFlag  string
		wantDelta *float64
	}{
		{name: "exactly at inclusive low", value: 3.5, wantFlag: "normal"},
		{name: "just below inclusive low", value: 3.499999, wantFlag: "low", wantDelta: f64(-0.000001)},
		{name: "exactly at inclusive high", value: 5.0, wantFlag: "normal"},
		{name: "just above inclusive high", value: 5.000001, wantFlag: "high", wantDelta: f64(0.000001)},
		{name: "well within", value: 4.2, wantFlag: "normal"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := evaluate(t, numeric(c.value, ""), inclusiveRange, nil)
			if got.GetFlag() != c.wantFlag {
				t.Fatalf("flag = %q, want %q (reason: %s)", got.GetFlag(), c.wantFlag, got.GetReason())
			}
			if c.wantDelta != nil {
				// Tolerance-based, not exact equality: *c.wantDelta is a Go
				// untyped-constant computation (arbitrary precision) while
				// got.DeltaFromBound is computed at runtime in float64 —
				// the two can differ in the last bit even though both are
				// "the same" subtraction, which is exactly the float
				// fencepost trap this package exists to avoid getting
				// wrong. The FLAG comparison above (not this delta check)
				// is the actual boundary-correctness assertion.
				if got.DeltaFromBound == nil || !almostEqual(*got.DeltaFromBound, *c.wantDelta, 1e-9) {
					t.Fatalf("delta = %v, want ~%v", got.DeltaFromBound, *c.wantDelta)
				}
			} else if got.GetFlag() == "normal" && got.DeltaFromBound != nil {
				t.Fatalf("delta should be absent for normal, got %v", *got.DeltaFromBound)
			}
			wantWithin := c.wantFlag == "normal"
			if got.Within == nil || *got.Within != wantWithin {
				t.Fatalf("within = %v, want %v", got.Within, wantWithin)
			}
		})
	}

	// Exclusive-boundary variants: "<200" (high exclusive) and ">40" (low
	// exclusive), each tested exactly-at, just-inside, just-outside.
	exclusiveHigh := rng(nil, f64(200), false, false, "")
	for _, c := range []struct {
		name     string
		value    float64
		wantFlag string
	}{
		{"exactly at exclusive high is OUT", 200, "high"},
		{"just below exclusive high is normal", 199.999, "normal"},
		{"just above exclusive high is high", 200.001, "high"},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := evaluate(t, numeric(c.value, ""), exclusiveHigh, nil)
			if got.GetFlag() != c.wantFlag {
				t.Fatalf("flag = %q, want %q", got.GetFlag(), c.wantFlag)
			}
		})
	}

	inclusiveHigh := rng(nil, f64(200), false, true, "")
	if got := evaluate(t, numeric(200, ""), inclusiveHigh, nil); got.GetFlag() != "normal" {
		t.Fatalf("inclusive high at exactly 200 should be normal, got %q", got.GetFlag())
	}

	exclusiveLow := rng(f64(40), nil, false, false, "")
	for _, c := range []struct {
		name     string
		value    float64
		wantFlag string
	}{
		{"exactly at exclusive low is OUT", 40, "low"},
		{"just above exclusive low is normal", 40.001, "normal"},
		{"just below exclusive low is low", 39.999, "low"},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := evaluate(t, numeric(c.value, ""), exclusiveLow, nil)
			if got.GetFlag() != c.wantFlag {
				t.Fatalf("flag = %q, want %q", got.GetFlag(), c.wantFlag)
			}
		})
	}

	inclusiveLow := rng(f64(40), nil, true, false, "")
	if got := evaluate(t, numeric(40, ""), inclusiveLow, nil); got.GetFlag() != "normal" {
		t.Fatalf("inclusive low at exactly 40 should be normal, got %q", got.GetFlag())
	}
}

// TestEvaluateResult_Censored proves censored values are compared by their
// possible-interval, never coerced to their threshold: a threshold at or
// beyond the reference bound is determinable, one that straddles it is
// indeterminate.
func TestEvaluateResult_Censored(t *testing.T) {
	r := rng(f64(3.5), f64(5.0), true, true, "")

	cases := []struct {
		name     string
		value    *gen.ParsedValue
		wantFlag string
	}{
		{"censored_low well below range is determinable low", censoredLow(0.5, ""), "low"},
		{"censored_low threshold exactly at inclusive low bound is determinable low", censoredLow(3.5, ""), "low"},
		{"censored_low threshold above low bound straddles: indeterminate", censoredLow(4.0, ""), "indeterminate"},
		{"censored_high well above range is determinable high", censoredHigh(1000, ""), "high"},
		{"censored_high threshold exactly at inclusive high bound is determinable high", censoredHigh(5.0, ""), "high"},
		{"censored_high threshold below high bound straddles: indeterminate", censoredHigh(4.0, ""), "indeterminate"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := evaluate(t, c.value, r, nil)
			if got.GetFlag() != c.wantFlag {
				t.Fatalf("flag = %q, want %q (reason: %s)", got.GetFlag(), c.wantFlag, got.GetReason())
			}
			// Censored kinds never carry a delta_from_bound, even when determinable.
			if got.DeltaFromBound != nil {
				t.Fatalf("censored value should never carry delta_from_bound, got %v", *got.DeltaFromBound)
			}
		})
	}
}

// TestEvaluateResult_ReportedRangeValue exercises a result reported as an
// interval itself (kind "range"), which is only classifiable when the
// WHOLE interval falls on one side of a boundary.
func TestEvaluateResult_ReportedRangeValue(t *testing.T) {
	r := rng(f64(3.5), f64(5.0), true, true, "")

	cases := []struct {
		name     string
		lo, hi   float64
		wantFlag string
	}{
		{"entirely within", 3.6, 4.9, "normal"},
		{"entirely below", 2.0, 3.0, "low"},
		{"entirely above", 6.0, 7.0, "high"},
		{"straddles the high boundary", 4.0, 6.0, "indeterminate"},
		{"straddles the low boundary", 3.0, 4.0, "indeterminate"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := evaluate(t, rangeValue(c.lo, c.hi, ""), r, nil)
			if got.GetFlag() != c.wantFlag {
				t.Fatalf("flag = %q, want %q (reason: %s)", got.GetFlag(), c.wantFlag, got.GetReason())
			}
		})
	}
}

// TestEvaluateResult_Indeterminate covers every documented indeterminate
// path other than censored-straddling (covered above): qualitative vs
// numeric range, and unit mismatch.
func TestEvaluateResult_Indeterminate(t *testing.T) {
	r := rng(f64(3.5), f64(5.0), true, true, "mg/dL")

	t.Run("qualitative value never compared to numeric range", func(t *testing.T) {
		got := evaluate(t, qualitative("POSITIVE"), r, nil)
		if got.GetFlag() != "indeterminate" {
			t.Fatalf("flag = %q, want indeterminate", got.GetFlag())
		}
		if got.Within != nil {
			t.Fatalf("within should be absent for indeterminate, got %v", *got.Within)
		}
	})

	t.Run("unit mismatch is indeterminate even when numerically it would be normal", func(t *testing.T) {
		got := evaluate(t, numeric(4.0, "mmol/L"), r, nil)
		if got.GetFlag() != "indeterminate" {
			t.Fatalf("flag = %q, want indeterminate", got.GetFlag())
		}
	})

	t.Run("matching units (after micro-sign canonicalization) do NOT mismatch", func(t *testing.T) {
		r2 := rng(f64(3.5), f64(5.0), true, true, "µg/dL")
		got := evaluate(t, numeric(4.0, "μg/dL"), r2, nil)
		if got.GetFlag() != "normal" {
			t.Fatalf("flag = %q, want normal (units should have canonicalized equal)", got.GetFlag())
		}
	})
}

// TestEvaluateResult_CriticalRangeTakesPrecedence proves a critical range,
// when it determines an outcome, wins over the plain low/high verdict.
func TestEvaluateResult_CriticalRangeTakesPrecedence(t *testing.T) {
	r := rng(f64(3.5), f64(5.0), true, true, "")
	critical := rng(f64(2.0), f64(8.0), true, true, "")

	if got := evaluate(t, numeric(1.0, ""), r, critical); got.GetFlag() != "critical_low" {
		t.Fatalf("flag = %q, want critical_low", got.GetFlag())
	}
	if got := evaluate(t, numeric(1.0, ""), r, nil); got.GetFlag() != "low" {
		t.Fatalf("without a critical range, flag = %q, want low", got.GetFlag())
	}
	if got := evaluate(t, numeric(9.0, ""), r, critical); got.GetFlag() != "critical_high" {
		t.Fatalf("flag = %q, want critical_high", got.GetFlag())
	}
	if got := evaluate(t, numeric(4.0, ""), r, critical); got.GetFlag() != "normal" {
		t.Fatalf("flag = %q, want normal", got.GetFlag())
	}
	// Between the normal and critical bounds: outside normal range but not
	// yet critical.
	if got := evaluate(t, numeric(3.0, ""), r, critical); got.GetFlag() != "low" {
		t.Fatalf("flag = %q, want low (below normal but not below critical)", got.GetFlag())
	}
}

// TestEvaluateResult_UpstreamParseErrorRefused proves EvaluateResult refuses
// to guess when its own input already carries a parse error.
func TestEvaluateResult_UpstreamParseErrorRefused(t *testing.T) {
	badValue := &gen.ParsedValue{Error: &gen.Error{Code: "INVALID_INPUT", Message: "boom"}}
	got := evaluate(t, badValue, rng(f64(1), f64(2), true, true, ""), nil)
	if got.GetError() == nil {
		t.Fatalf("expected an Error when the input value already carried one, got %+v", got)
	}
	if got.GetFlag() != "" {
		t.Fatalf("flag should be unset when evaluation could not run, got %q", got.GetFlag())
	}
}

// TestEvaluateResult_EndToEndWithRealParsers chains the actual parse nodes
// into EvaluateResult, proving the three nodes compose correctly rather
// than only testing hand-built messages.
func TestEvaluateResult_EndToEndWithRealParsers(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	value, err := nodes.ParseResultValue(ctx, ax, &gen.ParseResultValueRequest{Text: "<0.5 mg/dL"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := nodes.ParseReferenceRange(ctx, ax, &gen.ParseReferenceRangeRequest{Text: "3.5-5.0 mg/dL"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := nodes.EvaluateResult(ctx, ax, &gen.EvaluateResultRequest{Value: value, Range: r})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetFlag() != "low" {
		t.Fatalf("flag = %q, want low (reason: %s)", got.GetFlag(), got.GetReason())
	}
}
