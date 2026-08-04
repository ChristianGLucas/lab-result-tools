package nodes

import (
	"context"

	"christiangeorgelucas/lab-result-tools/axiom"
	gen "christiangeorgelucas/lab-result-tools/gen"
)

// BatchEvaluate validates and evaluates many lab result records in one
// call, each against its own reference range, returning a per-record flag
// plus aggregate counts. A record that fails validation is never evaluated
// — its EvaluationResult is "indeterminate" with a reason naming the
// validation failure, and it is counted separately as `invalid`.
func BatchEvaluate(ctx context.Context, ax axiom.Context, input *gen.BatchEvaluateRequest) (*gen.BatchEvaluateResult, error) {
	summary := &gen.BatchSummary{}
	items := make([]*gen.BatchResultItem, 0, len(input.GetRecords()))

	for _, br := range input.GetRecords() {
		summary.Total++
		validation := validateRecord(br.GetRecord())

		var evaluation *gen.EvaluationResult
		if !validation.GetValid() {
			evaluation = &gen.EvaluationResult{Flag: "indeterminate", Reason: "record failed validation and was not evaluated"}
			summary.Invalid++
		} else {
			evaluation = evaluateCore(validation.GetParsedValue(), validation.GetParsedRange(), nil)
			tallyFlag(summary, evaluation.GetFlag())
		}

		items = append(items, &gen.BatchResultItem{
			RecordId:   br.GetRecordId(),
			Validation: validation,
			Evaluation: evaluation,
		})
	}

	return &gen.BatchEvaluateResult{Items: items, Summary: summary}, nil
}

// tallyFlag increments the BatchSummary counter matching an evaluation's
// flag. A flag this package didn't itself produce (should not happen) is
// silently uncounted rather than crashing the batch.
func tallyFlag(summary *gen.BatchSummary, flag string) {
	switch flag {
	case "normal":
		summary.Normal++
	case "low":
		summary.Low++
	case "high":
		summary.High++
	case "critical_low":
		summary.CriticalLow++
	case "critical_high":
		summary.CriticalHigh++
	case "indeterminate":
		summary.Indeterminate++
	}
}
