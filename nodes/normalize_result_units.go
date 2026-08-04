package nodes

import (
	"context"

	"christiangeorgelucas/lab-result-tools/axiom"
	gen "christiangeorgelucas/lab-result-tools/gen"
)

// NormalizeResultUnits converts a lab concentration value between mass and
// molar units (e.g. "mg/dL" <-> "mmol/L"). It handles ONLY the
// mass/molar-per-volume vocabulary documented on NormalizeResultUnitsRequest
// — general physical-unit conversion belongs to christiangeorgelucas/
// unit-tools' Convert node instead. A mass<->molar conversion requires the
// analyte's molar mass as an explicit input; this node never assumes or
// looks one up.
func NormalizeResultUnits(ctx context.Context, ax axiom.Context, input *gen.NormalizeResultUnitsRequest) (*gen.NormalizedValue, error) {
	fromUnit, err := parseConcUnit(input.GetFromUnit())
	if err != nil {
		return &gen.NormalizedValue{Error: &gen.Error{Code: "UNSUPPORTED_UNIT", Message: "from_unit: " + err.Error()}}, nil
	}
	toUnit, err := parseConcUnit(input.GetToUnit())
	if err != nil {
		return &gen.NormalizedValue{Error: &gen.Error{Code: "UNSUPPORTED_UNIT", Message: "to_unit: " + err.Error()}}, nil
	}

	if fromUnit.kind != toUnit.kind && input.MolarMassGPerMol == nil {
		return &gen.NormalizedValue{Error: &gen.Error{
			Code:    "MOLAR_MASS_REQUIRED",
			Message: "converting between a mass unit and a molar unit requires molar_mass_g_per_mol",
		}}, nil
	}

	converted, err := convertConcentration(input.GetValue(), fromUnit, toUnit, input.MolarMassGPerMol)
	if err != nil {
		return &gen.NormalizedValue{Error: &gen.Error{Code: "INVALID_INPUT", Message: err.Error()}}, nil
	}

	return &gen.NormalizedValue{Value: converted, Unit: canonicalizeMicro(input.GetToUnit())}, nil
}
