package nodes

import (
	"context"
	"fmt"

	"christiangeorgelucas/lab-result-tools/axiom"
	gen "christiangeorgelucas/lab-result-tools/gen"
)

// EvaluateResult compares a parsed result value against a parsed reference
// range (and an optional tighter critical range), returning
// low/normal/high/critical_low/critical_high, or "indeterminate" when the
// comparison cannot be soundly decided. It never guesses a pass/fail.
func EvaluateResult(ctx context.Context, ax axiom.Context, input *gen.EvaluateResultRequest) (*gen.EvaluationResult, error) {
	return evaluateCore(input.GetValue(), input.GetRange(), input.CriticalRange), nil
}

// evaluateCore is the shared evaluation logic used by both EvaluateResult
// and BatchEvaluate.
func evaluateCore(value *gen.ParsedValue, rng *gen.ParsedRange, critical *gen.ParsedRange) *gen.EvaluationResult {
	if value.GetError() != nil {
		return &gen.EvaluationResult{Error: &gen.Error{Code: "INVALID_INPUT", Message: "value carries a parse error and cannot be evaluated"}}
	}
	if rng.GetError() != nil {
		return &gen.EvaluationResult{Error: &gen.Error{Code: "INVALID_INPUT", Message: "range carries a parse error and cannot be evaluated"}}
	}
	if critical != nil && critical.GetError() != nil {
		return &gen.EvaluationResult{Error: &gen.Error{Code: "INVALID_INPUT", Message: "critical_range carries a parse error and cannot be evaluated"}}
	}

	if value.GetKind() == "qualitative" {
		return indeterminate("qualitative result value %q cannot be compared to a numeric reference range", value.GetQualitativeText())
	}

	if mismatch := unitMismatchReason(value, rng, critical); mismatch != "" {
		return indeterminate("%s", mismatch)
	}

	iv, ok := valueInterval(value)
	if !ok {
		return &gen.EvaluationResult{Error: &gen.Error{Code: "INVALID_INPUT", Message: fmt.Sprintf("unrecognized ParsedValue.kind %q", value.GetKind())}}
	}

	// Critical range takes precedence when it determines an outcome — it is
	// the more specific, more severe bound.
	if critical != nil {
		if low := critical.Low; low != nil && iv.entirelyBelow(*low, critical.GetLowInclusive()) {
			return determinable("critical_low", value, low,
				fmt.Sprintf("value's possible range is entirely below the critical-low bound %v (%s)", *low, inclusiveWord(critical.GetLowInclusive())))
		}
		if high := critical.High; high != nil && iv.entirelyAbove(*high, critical.GetHighInclusive()) {
			return determinable("critical_high", value, high,
				fmt.Sprintf("value's possible range is entirely above the critical-high bound %v (%s)", *high, inclusiveWord(critical.GetHighInclusive())))
		}
	}

	if low := rng.Low; low != nil && iv.entirelyBelow(*low, rng.GetLowInclusive()) {
		return determinable("low", value, low,
			fmt.Sprintf("value's possible range is entirely below the reference low bound %v (%s)", *low, inclusiveWord(rng.GetLowInclusive())))
	}
	if high := rng.High; high != nil && iv.entirelyAbove(*high, rng.GetHighInclusive()) {
		return determinable("high", value, high,
			fmt.Sprintf("value's possible range is entirely above the reference high bound %v (%s)", *high, inclusiveWord(rng.GetHighInclusive())))
	}
	if iv.entirelyWithin(rng.Low, rng.GetLowInclusive(), rng.High, rng.GetHighInclusive()) {
		within := true
		return &gen.EvaluationResult{Flag: "normal", Within: &within, Reason: "value's possible range is entirely within the reference range"}
	}

	return indeterminate("value's possible range straddles a reference-range boundary; cannot be classified without more information")
}

// determinable builds the EvaluationResult for a bound that WAS violated.
// delta_from_bound is only meaningful for a single numeric point.
func determinable(flag string, value *gen.ParsedValue, bound *float64, reason string) *gen.EvaluationResult {
	within := false
	out := &gen.EvaluationResult{Flag: flag, Within: &within, Reason: reason}
	if value.GetKind() == "numeric" && value.Value != nil {
		delta := value.GetValue() - *bound
		out.DeltaFromBound = &delta
	}
	return out
}

func indeterminate(format string, args ...interface{}) *gen.EvaluationResult {
	return &gen.EvaluationResult{Flag: "indeterminate", Reason: fmt.Sprintf(format, args...)}
}

func inclusiveWord(inclusive bool) string {
	if inclusive {
		return "inclusive"
	}
	return "exclusive"
}

// valueInterval converts a ParsedValue into the interval of numbers it
// could possibly represent.
func valueInterval(value *gen.ParsedValue) (interval, bool) {
	switch value.GetKind() {
	case "numeric":
		return pointInterval(value.GetValue()), true
	case "censored_low":
		return rayBelow(value.GetValue()), true
	case "censored_high":
		return rayAbove(value.GetValue()), true
	case "range":
		return closedInterval(value.GetLow(), value.GetHigh()), true
	default:
		return interval{}, false
	}
}

// unitMismatchReason returns a non-empty reason if the value's unit, the
// range's unit, and the critical range's unit (when present) don't all
// agree — comparing only units that are actually non-empty, since an empty
// unit means "not specified" rather than "dimensionless".
func unitMismatchReason(value *gen.ParsedValue, rng *gen.ParsedRange, critical *gen.ParsedRange) string {
	type labeledUnit struct{ label, unit string }
	var pairs []labeledUnit
	add := func(label, u string) {
		if u != "" {
			pairs = append(pairs, labeledUnit{label, canonicalizeMicro(u)})
		}
	}
	add("value", value.GetUnit())
	add("range", rng.GetUnit())
	if critical != nil {
		add("critical_range", critical.GetUnit())
	}
	for i := 1; i < len(pairs); i++ {
		if pairs[i].unit != pairs[0].unit {
			return fmt.Sprintf("unit mismatch: %s is %q but %s is %q", pairs[0].label, pairs[0].unit, pairs[i].label, pairs[i].unit)
		}
	}
	return ""
}
