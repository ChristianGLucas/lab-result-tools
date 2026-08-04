package nodes

import (
	"context"
	"fmt"
	"strings"

	"christiangeorgelucas/lab-result-tools/axiom"
	gen "christiangeorgelucas/lab-result-tools/gen"
)

// ParseResultValue reads a messy lab-result string — "<0.5", ">1000",
// "5.2 mg/dL", "NEGATIVE", "1,234.5", "3.4-3.9" — into a typed value. A
// censored value ("<0.5"/">1000") is recorded as a threshold with a
// direction, never coerced to the number it names; a value that doesn't
// parse as numeric, censored, or a range falls back to "qualitative" with
// its trimmed, upper-cased text.
func ParseResultValue(ctx context.Context, ax axiom.Context, input *gen.ParseResultValueRequest) (*gen.ParsedValue, error) {
	return parseResultValueText(input.GetText()), nil
}

func parseResultValueText(original string) *gen.ParsedValue {
	trimmed := strings.TrimSpace(original)
	if trimmed == "" {
		return &gen.ParsedValue{
			Original: original,
			Error:    &gen.Error{Code: "INVALID_INPUT", Message: "result text is empty"},
		}
	}
	text := canonicalizeMicro(trimmed)

	// Censored values. ParseResultValue recognizes only "<"/">" (not
	// "<="/">="), matching the source shapes named in the package spec — a
	// censored RESULT reading has no independently meaningful
	// inclusive/exclusive nuance the way a reference-range BOUND does (see
	// ParseReferenceRange, which supports all four operators).
	for _, op := range []struct {
		prefix string
		kind   string
	}{
		{"<", "censored_low"},
		{">", "censored_high"},
	} {
		if !strings.HasPrefix(text, op.prefix) {
			continue
		}
		rest := strings.TrimSpace(text[len(op.prefix):])
		numTok, unit, ok := splitNumberAndUnit(rest)
		if !ok {
			return &gen.ParsedValue{
				Original: original,
				Error:    &gen.Error{Code: "INVALID_INPUT", Message: fmt.Sprintf("%q is not followed by a number", op.prefix)},
			}
		}
		v, err := parseDecimalToken(numTok)
		if err != nil {
			return &gen.ParsedValue{
				Original: original,
				Error:    &gen.Error{Code: "INVALID_INPUT", Message: fmt.Sprintf("censored threshold is not a valid number: %v", err)},
			}
		}
		return &gen.ParsedValue{Kind: op.kind, Value: &v, Unit: unit, Original: original}
	}

	// A reported value range, e.g. "3.4-3.9".
	if lo, hi, unit, ok := splitRange(text); ok {
		if lo > hi {
			return &gen.ParsedValue{
				Original: original,
				Error:    &gen.Error{Code: "RANGE_INVALID", Message: fmt.Sprintf("reported range low (%v) exceeds high (%v)", lo, hi)},
			}
		}
		return &gen.ParsedValue{Kind: "range", Low: &lo, High: &hi, Unit: unit, Original: original}
	}

	// A plain numeric value, with an optional unit.
	if numTok, unit, ok := splitNumberAndUnit(text); ok {
		if v, err := parseDecimalToken(numTok); err == nil {
			return &gen.ParsedValue{Kind: "numeric", Value: &v, Unit: unit, Original: original}
		}
	}

	// Anything else is a qualitative reading (e.g. "NEGATIVE", "TRACE").
	return &gen.ParsedValue{Kind: "qualitative", QualitativeText: strings.ToUpper(text), Original: original}
}
