package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/lab-result-tools/gen"
	"christiangeorgelucas/lab-result-tools/nodes"
)

func validateRecord(t *testing.T, record *gen.LabResultRecord) *gen.ValidationResult {
	t.Helper()
	got, err := nodes.ValidateResultRecord(context.Background(), newTestContext(t), &gen.ValidateResultRecordRequest{Record: record})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	return got
}

func hasIssue(issues []*gen.FieldIssue, field, severity string) bool {
	for _, i := range issues {
		if i.GetField() == field && i.GetSeverity() == severity {
			return true
		}
	}
	return false
}

func TestValidateResultRecord_CleanRecord(t *testing.T) {
	got := validateRecord(t, &gen.LabResultRecord{
		TestName:           "Glucose",
		ValueText:          "95",
		ReferenceRangeText: "70-110",
		CollectedAt:        "2026-08-03T14:30:00Z",
	})
	if !got.GetValid() {
		t.Fatalf("expected valid, got issues: %+v", got.GetIssues())
	}
	if got.GetParsedValue().GetKind() != "numeric" || got.GetParsedValue().GetValue() != 95 {
		t.Fatalf("parsed_value = %+v, want numeric 95", got.GetParsedValue())
	}
	if got.GetParsedRange().GetLow() != 70 || got.GetParsedRange().GetHigh() != 110 {
		t.Fatalf("parsed_range = %+v, want [70,110]", got.GetParsedRange())
	}
}

func TestValidateResultRecord_FieldIssues(t *testing.T) {
	t.Run("empty test_name is an error", func(t *testing.T) {
		got := validateRecord(t, &gen.LabResultRecord{ValueText: "5", ReferenceRangeText: "1-10"})
		if got.GetValid() {
			t.Fatalf("expected invalid")
		}
		if !hasIssue(got.GetIssues(), "test_name", "error") {
			t.Fatalf("expected a test_name error, got %+v", got.GetIssues())
		}
	})

	t.Run("malformed value_text is an error", func(t *testing.T) {
		got := validateRecord(t, &gen.LabResultRecord{TestName: "X", ValueText: "", ReferenceRangeText: "1-10"})
		if got.GetValid() {
			t.Fatalf("expected invalid")
		}
		if !hasIssue(got.GetIssues(), "value_text", "error") {
			t.Fatalf("expected a value_text error, got %+v", got.GetIssues())
		}
	})

	t.Run("reversed reference range is an error", func(t *testing.T) {
		got := validateRecord(t, &gen.LabResultRecord{TestName: "X", ValueText: "5", ReferenceRangeText: "10-1"})
		if got.GetValid() {
			t.Fatalf("expected invalid")
		}
		if !hasIssue(got.GetIssues(), "reference_range_text", "error") {
			t.Fatalf("expected a reference_range_text error, got %+v", got.GetIssues())
		}
	})

	t.Run("empty collected_at is only a warning", func(t *testing.T) {
		got := validateRecord(t, &gen.LabResultRecord{TestName: "X", ValueText: "5", ReferenceRangeText: "1-10", CollectedAt: ""})
		if !got.GetValid() {
			t.Fatalf("empty collected_at should not invalidate the record, got issues: %+v", got.GetIssues())
		}
		if !hasIssue(got.GetIssues(), "collected_at", "warning") {
			t.Fatalf("expected a collected_at warning, got %+v", got.GetIssues())
		}
	})

	t.Run("malformed collected_at is an error", func(t *testing.T) {
		got := validateRecord(t, &gen.LabResultRecord{TestName: "X", ValueText: "5", ReferenceRangeText: "1-10", CollectedAt: "not-a-date"})
		if got.GetValid() {
			t.Fatalf("expected invalid")
		}
		if !hasIssue(got.GetIssues(), "collected_at", "error") {
			t.Fatalf("expected a collected_at error, got %+v", got.GetIssues())
		}
	})

	t.Run("disagreeing unit field is a warning, not an error", func(t *testing.T) {
		got := validateRecord(t, &gen.LabResultRecord{
			TestName: "X", ValueText: "5 mg/dL", Unit: "mmol/L", ReferenceRangeText: "1-10",
		})
		if !got.GetValid() {
			t.Fatalf("disagreeing unit should be a warning only, got issues: %+v", got.GetIssues())
		}
		if !hasIssue(got.GetIssues(), "unit", "warning") {
			t.Fatalf("expected a unit warning, got %+v", got.GetIssues())
		}
	})
}
