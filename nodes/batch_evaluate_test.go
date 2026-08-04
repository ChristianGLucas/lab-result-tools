package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/lab-result-tools/gen"
	"christiangeorgelucas/lab-result-tools/nodes"
)

func TestBatchEvaluate_MixedPanel(t *testing.T) {
	req := &gen.BatchEvaluateRequest{Records: []*gen.BatchRecord{
		{RecordId: "normal", Record: &gen.LabResultRecord{TestName: "Glucose", ValueText: "95", ReferenceRangeText: "70-110"}},
		{RecordId: "low", Record: &gen.LabResultRecord{TestName: "Glucose", ValueText: "50", ReferenceRangeText: "70-110"}},
		{RecordId: "high", Record: &gen.LabResultRecord{TestName: "Glucose", ValueText: "150", ReferenceRangeText: "70-110"}},
		{RecordId: "indeterminate_qualitative", Record: &gen.LabResultRecord{TestName: "Culture", ValueText: "POSITIVE", ReferenceRangeText: "1-10"}},
		{RecordId: "invalid_bad_value", Record: &gen.LabResultRecord{TestName: "X", ValueText: "", ReferenceRangeText: "1-10"}},
		{RecordId: "invalid_no_test_name", Record: &gen.LabResultRecord{ValueText: "5", ReferenceRangeText: "1-10"}},
	}}

	got, err := nodes.BatchEvaluate(context.Background(), newTestContext(t), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}

	byID := map[string]*gen.BatchResultItem{}
	for _, item := range got.GetItems() {
		byID[item.GetRecordId()] = item
	}

	want := map[string]string{
		"normal":                    "normal",
		"low":                       "low",
		"high":                      "high",
		"indeterminate_qualitative": "indeterminate",
		"invalid_bad_value":         "indeterminate",
		"invalid_no_test_name":      "indeterminate",
	}
	for id, wantFlag := range want {
		item, ok := byID[id]
		if !ok {
			t.Fatalf("missing item %q", id)
		}
		if got := item.GetEvaluation().GetFlag(); got != wantFlag {
			t.Fatalf("%s: flag = %q, want %q", id, got, wantFlag)
		}
	}

	if byID["invalid_bad_value"].GetValidation().GetValid() {
		t.Fatalf("invalid_bad_value should have Validation.Valid = false")
	}
	if byID["invalid_no_test_name"].GetValidation().GetValid() {
		t.Fatalf("invalid_no_test_name should have Validation.Valid = false")
	}

	s := got.GetSummary()
	if s.GetTotal() != 6 {
		t.Fatalf("total = %d, want 6", s.GetTotal())
	}
	if s.GetNormal() != 1 || s.GetLow() != 1 || s.GetHigh() != 1 {
		t.Fatalf("summary = %+v, want normal=1 low=1 high=1", s)
	}
	if s.GetIndeterminate() != 1 {
		t.Fatalf("indeterminate = %d, want 1 (only the valid-but-qualitative record)", s.GetIndeterminate())
	}
	if s.GetInvalid() != 2 {
		t.Fatalf("invalid = %d, want 2", s.GetInvalid())
	}
}

func TestBatchEvaluate_Empty(t *testing.T) {
	got, err := nodes.BatchEvaluate(context.Background(), newTestContext(t), &gen.BatchEvaluateRequest{})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if len(got.GetItems()) != 0 || got.GetSummary().GetTotal() != 0 {
		t.Fatalf("expected an empty result for an empty batch, got %+v", got)
	}
}
