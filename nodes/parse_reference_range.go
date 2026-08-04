package nodes

import (
	"context"
	"fmt"
	"strings"

	"christiangeorgelucas/lab-result-tools/axiom"
	gen "christiangeorgelucas/lab-result-tools/gen"
)

// ParseReferenceRange reads a reference-range string — "3.5-5.0", "<200",
// ">=40", "70 - 110 mg/dL" — into explicit low/high bounds with their own
// inclusive/exclusive flags: a bare "low-high" range is inclusive on BOTH
// ends; "<"/">" are EXCLUSIVE of the stated bound; "<="/">=" are INCLUSIVE
// of it. A reversed range (low > high) is rejected as RANGE_INVALID rather
// than silently accepted.
func ParseReferenceRange(ctx context.Context, ax axiom.Context, input *gen.ParseReferenceRangeRequest) (*gen.ParsedRange, error) {
	return parseReferenceRangeText(input.GetText()), nil
}

// rangeOperator pairs a comparison-operator prefix with whether it makes
// its bound inclusive, and whether that bound is the range's low or high
// side. Longer operators ("<=", ">=") must be checked before their
// single-character prefixes ("<", ">").
type rangeOperator struct {
	prefix    string
	isLow     bool
	inclusive bool
}

var rangeOperators = []rangeOperator{
	{"<=", false, true},
	{">=", true, true},
	{"<", false, false},
	{">", true, false},
}

func parseReferenceRangeText(original string) *gen.ParsedRange {
	trimmed := strings.TrimSpace(original)
	if trimmed == "" {
		return &gen.ParsedRange{
			Original: original,
			Error:    &gen.Error{Code: "INVALID_INPUT", Message: "reference-range text is empty"},
		}
	}
	text := canonicalizeMicro(trimmed)

	for _, op := range rangeOperators {
		if !strings.HasPrefix(text, op.prefix) {
			continue
		}
		rest := strings.TrimSpace(text[len(op.prefix):])
		numTok, unit, ok := splitNumberAndUnit(rest)
		if !ok {
			return &gen.ParsedRange{
				Original: original,
				Error:    &gen.Error{Code: "INVALID_INPUT", Message: fmt.Sprintf("%q is not followed by a number", op.prefix)},
			}
		}
		v, err := parseDecimalToken(numTok)
		if err != nil {
			return &gen.ParsedRange{
				Original: original,
				Error:    &gen.Error{Code: "INVALID_INPUT", Message: fmt.Sprintf("range bound is not a valid number: %v", err)},
			}
		}
		out := &gen.ParsedRange{Unit: unit, Original: original}
		if op.isLow {
			out.Low = &v
			out.LowInclusive = op.inclusive
		} else {
			out.High = &v
			out.HighInclusive = op.inclusive
		}
		return out
	}

	// A bare "low-high" range, inclusive on both ends.
	if lo, hi, unit, ok := splitRange(text); ok {
		if lo > hi {
			return &gen.ParsedRange{
				Original: original,
				Error:    &gen.Error{Code: "RANGE_INVALID", Message: fmt.Sprintf("reference range low (%v) exceeds high (%v)", lo, hi)},
			}
		}
		return &gen.ParsedRange{Low: &lo, High: &hi, LowInclusive: true, HighInclusive: true, Unit: unit, Original: original}
	}

	return &gen.ParsedRange{
		Original: original,
		Error:    &gen.Error{Code: "INVALID_INPUT", Message: "not a recognized reference-range expression"},
	}
}
