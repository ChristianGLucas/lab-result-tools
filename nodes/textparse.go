package nodes

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// canonicalizeMicro replaces U+00B5 (MICRO SIGN) with U+03BC (GREEK SMALL
// LETTER MU) so the two spellings of "micro" that lab systems mix compare
// equal. This is a targeted substitution, not general Unicode
// normalization — see the messages.proto file header for why that scope is
// deliberate.
func canonicalizeMicro(s string) string {
	return strings.ReplaceAll(s, "µ", "μ")
}

// numTokenPlain matches a signed or unsigned decimal number, optionally
// using comma thousands-grouping, e.g. "5.2", "-3", "1,234.5".
var numTokenPlain = regexp.MustCompile(`^[+-]?\d[\d,]*(?:\.\d+)?$`)

// numTokenGrouped matches ONLY the strict thousands-grouped form
// ("#" or "##" or "###" then repeated ",###" groups), so "1,23.4" (a
// malformed grouping) is rejected rather than silently accepted.
var numTokenGrouped = regexp.MustCompile(`^[+-]?\d{1,3}(,\d{3})+(\.\d+)?$`)

// numTokenSimple matches a plain number with no thousands separators.
var numTokenSimple = regexp.MustCompile(`^[+-]?\d+(\.\d+)?$`)

// parseDecimalToken parses a decimal-literal string into a finite float64,
// accepting comma thousands-grouping only in its strict form. It rejects
// anything else (rather than guessing at a malformed grouping like
// "1,23.4"), and rejects non-finite results.
func parseDecimalToken(tok string) (float64, error) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return 0, fmt.Errorf("empty numeric token")
	}
	if !numTokenPlain.MatchString(tok) {
		return 0, fmt.Errorf("not a decimal number: %q", tok)
	}
	var cleaned string
	switch {
	case numTokenGrouped.MatchString(tok):
		cleaned = strings.ReplaceAll(tok, ",", "")
	case numTokenSimple.MatchString(tok):
		cleaned = tok
	default:
		return 0, fmt.Errorf("malformed thousands grouping: %q", tok)
	}
	v, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("non-finite number: %q", tok)
	}
	return v, nil
}

// leadingNumberRe captures a leading signed decimal-literal token (allowing
// comma grouping) and the trimmed remainder of the string as a candidate
// unit expression.
var leadingNumberRe = regexp.MustCompile(`^\s*([+-]?\d[\d,]*(?:\.\d+)?)\s*(.*)$`)

// splitNumberAndUnit splits a string into its leading numeric token and
// whatever text follows (trimmed), treated as a unit expression. It does
// not itself validate the numeric token — call parseDecimalToken on the
// result. Returns ok=false if the string does not begin with a digit or
// sign followed by a digit.
func splitNumberAndUnit(s string) (numTok string, unit string, ok bool) {
	m := leadingNumberRe.FindStringSubmatch(s)
	if m == nil {
		return "", "", false
	}
	return m[1], canonicalizeMicro(strings.TrimSpace(m[2])), true
}

// bareRangeRe matches "<low> - <high> [unit]" — two non-negative,
// non-signed decimal tokens separated by a bare hyphen (with optional
// surrounding whitespace), followed by an optional unit. Bounds are
// intentionally unsigned: a leading '-' is treated as a negative number's
// sign, not a range dash, so a lone negative number is parsed as numeric,
// never as a one-sided range.
var bareRangeRe = regexp.MustCompile(`^\s*(\d[\d,]*(?:\.\d+)?)\s*-\s*(\d[\d,]*(?:\.\d+)?)\s*(.*)$`)

// splitRange recognizes a bare "low-high [unit]" expression. ok is false if
// the string doesn't match that shape at all; a matching shape whose
// numbers fail to parse (should not happen given the regex) also returns
// ok=false so callers fall back to other interpretations.
func splitRange(s string) (lo, hi float64, unit string, ok bool) {
	m := bareRangeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, "", false
	}
	loVal, err := parseDecimalToken(m[1])
	if err != nil {
		return 0, 0, "", false
	}
	hiVal, err := parseDecimalToken(m[2])
	if err != nil {
		return 0, 0, "", false
	}
	return loVal, hiVal, canonicalizeMicro(strings.TrimSpace(m[3])), true
}
