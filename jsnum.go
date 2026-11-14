package lunr

import (
	"math"
	"strconv"
	"strings"
)

// jsNumberToString renders a float the way JS String(number) does: decimal
// notation for magnitudes in [1e-6, 1e21), exponent notation outside it.
// Shared verbatim with the fuse-search port; verified against Node.
func jsNumberToString(f float64) string {
	if f == 0 {
		if math.Signbit(f) {
			return "-0"
		}
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	s := strconv.FormatFloat(f, 'e', -1, 64)
	parts := strings.SplitN(s, "e", 2)
	exp := parts[1]
	sign := ""
	if strings.HasPrefix(exp, "+") || strings.HasPrefix(exp, "-") {
		sign = exp[:1]
		exp = exp[1:]
	}
	exp = strings.TrimLeft(exp, "0")
	if exp == "" {
		exp = "0"
	}
	if sign == "" {
		sign = "+"
	}
	return parts[0] + "e" + sign + exp
}
