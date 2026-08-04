package nodes

import (
	"fmt"
	"math"
	"strings"
)

// concUnit is a parsed lab concentration unit: a mass or molar amount,
// each expressed as a power-of-ten SI prefix over gram or mole, per a
// power-of-ten volume (a power of ten over liter).
type concUnit struct {
	kind      string // "mass" or "molar"
	prefixExp int    // power of ten relative to the base (g or mol)
	volExp    int    // power of ten relative to liter
}

// massPrefixExponents maps a mass SI prefix (as it appears immediately
// before "g") to its power of ten. The empty prefix is bare grams.
var massPrefixExponents = map[string]int{
	"":  0,
	"k": 3,
	"m": -3,
	"u": -6,
	"μ": -6,
	"n": -9,
	"p": -12,
}

// molarPrefixExponents maps a molar SI prefix (as it appears immediately
// before "mol") to its power of ten. The empty prefix is bare moles.
var molarPrefixExponents = map[string]int{
	"":  0,
	"m": -3,
	"u": -6,
	"μ": -6,
	"n": -9,
	"p": -12,
}

// volumePrefixExponents maps a volume prefix (as it appears immediately
// before "L") to its power of ten relative to one liter.
var volumePrefixExponents = map[string]int{
	"":  0,
	"d": -1,
	"c": -2,
	"m": -3,
	"u": -6,
	"μ": -6,
}

// parseConcUnit parses a lab concentration unit such as "mg/dL", "umol/L",
// "μg/mL" into its structured form. It accepts ONLY the mass-per-volume
// and molar-per-volume shapes this package documents; anything else
// (a bare mass, a molar unit with no volume, an unrecognized prefix, a
// unit with no "/") is rejected.
func parseConcUnit(raw string) (concUnit, error) {
	u := canonicalizeMicro(strings.TrimSpace(raw))
	if u == "" {
		return concUnit{}, fmt.Errorf("empty unit")
	}
	parts := strings.SplitN(u, "/", 2)
	if len(parts) != 2 {
		return concUnit{}, fmt.Errorf("not a <amount>/<volume> concentration unit: %q", raw)
	}
	amount, volume := parts[0], parts[1]

	var kind string
	var prefixExp int
	switch {
	case strings.HasSuffix(amount, "mol"):
		prefix := strings.TrimSuffix(amount, "mol")
		exp, ok := molarPrefixExponents[prefix]
		if !ok {
			return concUnit{}, fmt.Errorf("unrecognized molar prefix %q in %q", prefix, raw)
		}
		kind, prefixExp = "molar", exp
	case strings.HasSuffix(amount, "g"):
		prefix := strings.TrimSuffix(amount, "g")
		exp, ok := massPrefixExponents[prefix]
		if !ok {
			return concUnit{}, fmt.Errorf("unrecognized mass prefix %q in %q", prefix, raw)
		}
		kind, prefixExp = "mass", exp
	default:
		return concUnit{}, fmt.Errorf("unrecognized amount unit %q (want a mass unit ending in \"g\" or a molar unit ending in \"mol\")", amount)
	}

	var volPrefix string
	switch {
	case strings.HasSuffix(volume, "L"):
		volPrefix = strings.TrimSuffix(volume, "L")
	case strings.HasSuffix(volume, "l"):
		volPrefix = strings.TrimSuffix(volume, "l")
	default:
		return concUnit{}, fmt.Errorf("unrecognized volume unit %q (want a volume ending in \"L\")", volume)
	}
	volExp, ok := volumePrefixExponents[volPrefix]
	if !ok {
		return concUnit{}, fmt.Errorf("unrecognized volume prefix %q in %q", volPrefix, raw)
	}

	return concUnit{kind: kind, prefixExp: prefixExp, volExp: volExp}, nil
}

// convertConcentration converts value (expressed in the `from` unit) into
// the `to` unit. If from and to are different kinds (mass vs molar), the
// analyte's molar mass in grams per mole is required to bridge them; nil
// returns an error rather than assuming one.
func convertConcentration(value float64, from, to concUnit, molarMassGPerMol *float64) (float64, error) {
	// Canonicalize `from` to a per-liter amount of its own kind (grams/L or
	// moles/L), by undoing its prefix and volume scaling.
	canonicalPerLiter := value * math.Pow(10, float64(from.prefixExp-from.volExp))

	if from.kind == to.kind {
		// Same kind: scale straight from canonical per-liter to the
		// target's prefix/volume — no molar mass involved.
		return canonicalPerLiter * math.Pow(10, float64(to.volExp-to.prefixExp)), nil
	}

	if molarMassGPerMol == nil {
		return 0, fmt.Errorf("molar_mass_g_per_mol is required to convert between mass and molar units")
	}
	molarMass := *molarMassGPerMol
	if !(molarMass > 0) || math.IsInf(molarMass, 0) || math.IsNaN(molarMass) {
		return 0, fmt.Errorf("molar_mass_g_per_mol must be a finite positive number, got %v", molarMass)
	}

	var canonicalTargetKind float64 // per-liter amount in the TARGET's kind
	if from.kind == "mass" && to.kind == "molar" {
		// grams/L -> moles/L
		canonicalTargetKind = canonicalPerLiter / molarMass
	} else if from.kind == "molar" && to.kind == "mass" {
		// moles/L -> grams/L
		canonicalTargetKind = canonicalPerLiter * molarMass
	} else {
		return 0, fmt.Errorf("internal: unrecognized unit kinds %q -> %q", from.kind, to.kind)
	}

	return canonicalTargetKind * math.Pow(10, float64(to.volExp-to.prefixExp)), nil
}
