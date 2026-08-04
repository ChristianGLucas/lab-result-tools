package nodes

import "math"

// interval represents the set of values a ParsedValue could possibly be:
//   - a single numeric reading is a closed point [v, v]
//   - a censored_low "<t" reading is the open ray (-inf, t)
//   - a censored_high ">t" reading is the open ray (t, +inf)
//   - a reported value range is the closed interval [low, high]
//
// lo/hi may be +/-Inf. loOpen/hiOpen say whether lo/hi themselves are part
// of the interval (irrelevant when that end is infinite).
type interval struct {
	lo, hi         float64
	loOpen, hiOpen bool
}

func pointInterval(v float64) interval       { return interval{lo: v, hi: v} }
func rayBelow(t float64) interval            { return interval{lo: math.Inf(-1), hi: t, hiOpen: true} }
func rayAbove(t float64) interval            { return interval{lo: t, hi: math.Inf(1), loOpen: true} }
func closedInterval(lo, hi float64) interval { return interval{lo: lo, hi: hi} }

// entirelyBelow reports whether every value the interval could take is
// strictly outside a reference range on its low side — i.e. is guaranteed
// to fail "x > bound || (x == bound && boundInclusive)". This is decided
// purely from the interval's upper end, since that is the closest point to
// being "in range".
func (iv interval) entirelyBelow(bound float64, boundInclusive bool) bool {
	if math.IsInf(iv.hi, 1) {
		return false // extends to +inf: can never be entirely below a finite bound
	}
	if iv.hiOpen {
		// Every value is strictly < iv.hi, so iv.hi <= bound guarantees
		// every value is < bound (hence not in range), regardless of
		// boundInclusive.
		return iv.hi <= bound
	}
	return iv.hi < bound || (iv.hi == bound && !boundInclusive)
}

// entirelyAbove is the mirror of entirelyBelow for a reference range's high
// side, decided from the interval's lower end.
func (iv interval) entirelyAbove(bound float64, boundInclusive bool) bool {
	if math.IsInf(iv.lo, -1) {
		return false // extends to -inf: can never be entirely above a finite bound
	}
	if iv.loOpen {
		return iv.lo >= bound
	}
	return iv.lo > bound || (iv.lo == bound && !boundInclusive)
}

// entirelyWithin reports whether EVERY value the interval could take
// satisfies membership in [low, high] (either bound may be nil, meaning
// unbounded on that side).
func (iv interval) entirelyWithin(low *float64, lowInclusive bool, high *float64, highInclusive bool) bool {
	if low != nil {
		l := *low
		var okLow bool
		if iv.loOpen {
			okLow = iv.lo >= l
		} else {
			okLow = iv.lo > l || (iv.lo == l && lowInclusive)
		}
		if !okLow {
			return false
		}
	}
	if high != nil {
		h := *high
		var okHigh bool
		if iv.hiOpen {
			okHigh = iv.hi <= h
		} else {
			okHigh = iv.hi < h || (iv.hi == h && highInclusive)
		}
		if !okHigh {
			return false
		}
	}
	return true
}
