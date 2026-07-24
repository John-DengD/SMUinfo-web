package product

import (
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

// Price wraps pgtype.Numeric so it serializes to JSON exactly like a Java
// BigDecimal read from a NUMERIC(10,2) column: the scale is fixed at 2
// (e.g. 12.50, 99.00, 0.00), matching Jackson's default BigDecimal output
// byte-for-byte for the WeChat miniprogram wire contract.
//
// This is necessary because the Postgres binary numeric wire format strips
// trailing zero digits when pgx decodes a value, so a stored 12.50 would
// otherwise round-trip through pgtype.Numeric.MarshalJSON as "12.5". By
// re-scaling to 2 decimal places on output we restore the Java shape.
//
// For input (create/update request bodies) it delegates to pgtype so that a
// JSON number such as 12.5 is parsed correctly; the DB column enforces the
// (10,2) scale on write regardless.
type Price struct {
	pgtype.Numeric
}

const priceScale = 2

// MarshalJSON emits null when invalid, otherwise the value at fixed scale 2.
func (p Price) MarshalJSON() ([]byte, error) {
	if !p.Valid {
		return []byte("null"), nil
	}
	if p.NaN {
		return []byte(`"NaN"`), nil
	}
	return []byte(p.plainStringScale2()), nil
}

// UnmarshalJSON delegates to pgtype.Numeric (accepts JSON numbers/strings).
func (p *Price) UnmarshalJSON(src []byte) error {
	return p.Numeric.UnmarshalJSON(src)
}

// plainStringScale2 renders the numeric with exactly two fractional digits.
func (p Price) plainStringScale2() string {
	if p.Int == nil {
		return "0.00"
	}

	// unscaled = Int * 10^Exp, target scale 2 => value * 100.
	// scaled = Int * 10^(Exp + 2)
	scaled := new(big.Int).Set(p.Int)
	shift := int(p.Exp) + priceScale
	switch {
	case shift > 0:
		mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(shift)), nil)
		scaled.Mul(scaled, mul)
	case shift < 0:
		// Divide with rounding half-away-from-zero to land on 2 decimals.
		div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-shift)), nil)
		q, r := new(big.Int), new(big.Int)
		q.QuoRem(scaled, div, r)
		// round half up (away from zero) on the absolute remainder
		twice := new(big.Int).Abs(r)
		twice.Lsh(twice, 1)
		if twice.Cmp(div) >= 0 {
			if scaled.Sign() < 0 {
				q.Sub(q, big.NewInt(1))
			} else {
				q.Add(q, big.NewInt(1))
			}
		}
		scaled = q
	}

	neg := scaled.Sign() < 0
	abs := new(big.Int).Abs(scaled)
	digits := abs.String()
	for len(digits) < priceScale+1 {
		digits = "0" + digits
	}
	intPart := digits[:len(digits)-priceScale]
	fracPart := digits[len(digits)-priceScale:]
	out := intPart + "." + fracPart
	if neg {
		out = "-" + out
	}
	return out
}
