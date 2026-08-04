package nodes_test

import (
	"context"
	"math"
	"testing"

	gen "christiangeorgelucas/lab-result-tools/gen"
	"christiangeorgelucas/lab-result-tools/nodes"
)

// glucoseMolarMassGPerMol is D-glucose's molar mass, 180.16 g/mol
// (PubChem CID 5793, https://pubchem.ncbi.nlm.nih.gov/compound/5793,
// "Molecular Weight: 180.16 g/mol"). This is the INDEPENDENT-ORACLE value
// for TestNormalizeResultUnits_MolarMassConversion below — the test does
// not derive this number from the code under test.
const glucoseMolarMassGPerMol = 180.16

func normalize(t *testing.T, value float64, from, to string, molarMass *float64) *gen.NormalizedValue {
	t.Helper()
	got, err := nodes.NormalizeResultUnits(context.Background(), newTestContext(t), &gen.NormalizeResultUnitsRequest{
		Value: value, FromUnit: from, ToUnit: to, MolarMassGPerMol: molarMass,
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	return got
}

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// TestNormalizeResultUnits_MolarMassConversion hand-verifies a mass<->molar
// conversion against glucose's published molar mass. 100 mg/dL glucose is
// 1 g/L; divided by 180.16 g/mol that is 0.0055498... mol/L, i.e.
// 5.5498... mmol/L (the same arithmetic underlying the widely-cited
// mg/dL x 0.0555 = mmol/L glucose conversion factor). This is an
// independent oracle: computed here from the cited molar mass, not copied
// from the implementation.
func TestNormalizeResultUnits_MolarMassConversion(t *testing.T) {
	mm := glucoseMolarMassGPerMol
	// 100 mg/dL -> g/L: 100 mg = 0.1 g; "per dL" -> "per L" multiplies by
	// 10 (1 L = 10 dL), giving 0.1 * 10 = 1 g/L. Then g/L -> mol/L divides
	// by molar mass, and mol/L -> mmol/L multiplies by 1000.
	want := (100.0 / 1000.0) * 10.0 / mm * 1000.0
	got := normalize(t, 100, "mg/dL", "mmol/L", &mm)
	if got.GetError() != nil {
		t.Fatalf("unexpected error: %+v", got.GetError())
	}
	if !almostEqual(got.GetValue(), want, 1e-6) {
		t.Fatalf("100 mg/dL glucose -> mmol/L = %v, want %v (~5.55)", got.GetValue(), want)
	}

	// And the round trip back to mg/dL recovers the original value.
	back := normalize(t, got.GetValue(), "mmol/L", "mg/dL", &mm)
	if !almostEqual(back.GetValue(), 100, 1e-6) {
		t.Fatalf("round trip mmol/L -> mg/dL = %v, want ~100", back.GetValue())
	}
}

func TestNormalizeResultUnits_SameKindConversions(t *testing.T) {
	cases := []struct {
		name     string
		value    float64
		from, to string
		want     float64
	}{
		{"mg/dL to g/L", 100, "mg/dL", "g/L", 1.0},
		{"g/L to mg/dL", 1, "g/L", "mg/dL", 100.0},
		{"mmol/L to umol/L", 5, "mmol/L", "umol/L", 5000.0},
		{"mmol/L to μmol/L (greek mu)", 5, "mmol/L", "μmol/L", 5000.0},
		{"mmol/L to µmol/L (micro sign)", 5, "mmol/L", "µmol/L", 5000.0},
		{"ng/mL to ug/dL", 1000, "ng/mL", "ug/dL", 100.0},
		{"identity mg/dL to mg/dL", 42, "mg/dL", "mg/dL", 42.0},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := normalize(t, c.value, c.from, c.to, nil)
			if got.GetError() != nil {
				t.Fatalf("unexpected error: %+v", got.GetError())
			}
			if !almostEqual(got.GetValue(), c.want, 1e-9) {
				t.Fatalf("%v %s -> %s = %v, want %v", c.value, c.from, c.to, got.GetValue(), c.want)
			}
		})
	}
}

func TestNormalizeResultUnits_Errors(t *testing.T) {
	mm := glucoseMolarMassGPerMol
	badMM := -5.0
	zeroMM := 0.0

	t.Run("mass to molar without molar mass is MOLAR_MASS_REQUIRED", func(t *testing.T) {
		got := normalize(t, 100, "mg/dL", "mmol/L", nil)
		if got.GetError() == nil || got.GetError().GetCode() != "MOLAR_MASS_REQUIRED" {
			t.Fatalf("want MOLAR_MASS_REQUIRED, got %+v", got)
		}
	})
	t.Run("unsupported unit IU/mL is UNSUPPORTED_UNIT", func(t *testing.T) {
		got := normalize(t, 1, "IU/mL", "mg/dL", &mm)
		if got.GetError() == nil || got.GetError().GetCode() != "UNSUPPORTED_UNIT" {
			t.Fatalf("want UNSUPPORTED_UNIT, got %+v", got)
		}
	})
	t.Run("garbage unit is UNSUPPORTED_UNIT", func(t *testing.T) {
		got := normalize(t, 1, "banana", "mg/dL", &mm)
		if got.GetError() == nil || got.GetError().GetCode() != "UNSUPPORTED_UNIT" {
			t.Fatalf("want UNSUPPORTED_UNIT, got %+v", got)
		}
	})
	t.Run("negative molar mass is rejected", func(t *testing.T) {
		got := normalize(t, 100, "mg/dL", "mmol/L", &badMM)
		if got.GetError() == nil {
			t.Fatalf("want an error for a negative molar mass, got %+v", got)
		}
	})
	t.Run("zero molar mass is rejected", func(t *testing.T) {
		got := normalize(t, 100, "mg/dL", "mmol/L", &zeroMM)
		if got.GetError() == nil {
			t.Fatalf("want an error for a zero molar mass, got %+v", got)
		}
	})
	t.Run("bare mass with no volume is unsupported", func(t *testing.T) {
		got := normalize(t, 1, "mg", "mg/dL", &mm)
		if got.GetError() == nil || got.GetError().GetCode() != "UNSUPPORTED_UNIT" {
			t.Fatalf("want UNSUPPORTED_UNIT, got %+v", got)
		}
	})
}
