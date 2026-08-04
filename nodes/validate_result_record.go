package nodes

import (
	"context"
	"fmt"
	"time"

	"christiangeorgelucas/lab-result-tools/axiom"
	gen "christiangeorgelucas/lab-result-tools/gen"
)

// ValidateResultRecord checks one full lab result record (test name, value,
// unit, reference range, collection timestamp) and returns a field-level
// list of errors/warnings plus the record's already-parsed value and range.
func ValidateResultRecord(ctx context.Context, ax axiom.Context, input *gen.ValidateResultRecordRequest) (*gen.ValidationResult, error) {
	return validateRecord(input.GetRecord()), nil
}

// validateRecord is the shared validation logic used by both
// ValidateResultRecord and BatchEvaluate.
func validateRecord(record *gen.LabResultRecord) *gen.ValidationResult {
	var issues []*gen.FieldIssue
	addIssue := func(field, severity, message string) {
		issues = append(issues, &gen.FieldIssue{Field: field, Severity: severity, Message: message})
	}

	if record.GetTestName() == "" {
		addIssue("test_name", "error", "test_name is empty")
	}

	parsedValue := parseResultValueText(record.GetValueText())
	if err := parsedValue.GetError(); err != nil {
		addIssue("value_text", "error", fmt.Sprintf("%s: %s", err.GetCode(), err.GetMessage()))
	}

	parsedRange := parseReferenceRangeText(record.GetReferenceRangeText())
	if err := parsedRange.GetError(); err != nil {
		addIssue("reference_range_text", "error", fmt.Sprintf("%s: %s", err.GetCode(), err.GetMessage()))
	}

	// A unit reported separately from value_text/reference_range_text, and
	// disagreeing with a unit already embedded in one of them, is worth a
	// warning — it does not by itself invalidate the record, since the
	// separate `unit` field's relationship to the embedded ones is a
	// source-system convention, not something this package can adjudicate.
	if u := record.GetUnit(); u != "" {
		if pv := canonicalizeMicro(u); parsedValue.GetUnit() != "" && parsedValue.GetUnit() != pv {
			addIssue("unit", "warning", fmt.Sprintf("record.unit (%q) disagrees with the unit embedded in value_text (%q)", u, parsedValue.GetUnit()))
		}
	}

	switch record.GetCollectedAt() {
	case "":
		addIssue("collected_at", "warning", "collected_at is empty")
	default:
		if _, err := time.Parse(time.RFC3339, record.GetCollectedAt()); err != nil {
			addIssue("collected_at", "error", fmt.Sprintf("collected_at is not a valid RFC3339 timestamp: %v", err))
		}
	}

	valid := true
	for _, issue := range issues {
		if issue.GetSeverity() == "error" {
			valid = false
			break
		}
	}

	return &gen.ValidationResult{
		Valid:       valid,
		Issues:      issues,
		ParsedValue: parsedValue,
		ParsedRange: parsedRange,
	}
}
