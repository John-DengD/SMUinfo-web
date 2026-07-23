package product

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestPriceMarshalFixedScale2 verifies Price always emits exactly two decimal
// places to match Java BigDecimal-from-NUMERIC(10,2) / Jackson output, including
// the crucial case where pgx decodes a stored 12.50 as {Int:125, Exp:-1} (the
// Postgres binary wire format strips the trailing zero).
func TestPriceMarshalFixedScale2(t *testing.T) {
	cases := []struct {
		name string
		p    Price
		want string
	}{
		// DB-decoded value with trailing zero stripped -> must restore "12.50".
		{"exp-1 trailing zero stripped", Price{pgtype.Numeric{Int: big.NewInt(125), Exp: -1, Valid: true}}, "12.50"},
		{"exp-2 exact", Price{pgtype.Numeric{Int: big.NewInt(1250), Exp: -2, Valid: true}}, "12.50"},
		{"whole number", Price{pgtype.Numeric{Int: big.NewInt(99), Exp: 0, Valid: true}}, "99.00"},
		{"exp positive", Price{pgtype.Numeric{Int: big.NewInt(5), Exp: 1, Valid: true}}, "50.00"},
		{"zero", Price{pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true}}, "0.00"},
		{"sub-one", Price{pgtype.Numeric{Int: big.NewInt(5), Exp: -1, Valid: true}}, "0.50"},
		{"more than 2 decimals rounds", Price{pgtype.Numeric{Int: big.NewInt(12345), Exp: -3, Valid: true}}, "12.35"},
		{"large value", Price{pgtype.Numeric{Int: big.NewInt(9999999), Exp: -2, Valid: true}}, "99999.99"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.p)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.want {
				t.Fatalf("Price marshal = %s, want %s", b, tc.want)
			}
		})
	}
}

func TestPriceMarshalNull(t *testing.T) {
	var p Price // invalid
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "null" {
		t.Fatalf("invalid Price should marshal null, got %s", b)
	}
}

func TestPriceUnmarshalRoundTrip(t *testing.T) {
	var p Price
	if err := json.Unmarshal([]byte("12.5"), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Valid {
		t.Fatal("expected valid after unmarshal")
	}
	// Even though input had one decimal, output is normalized to scale 2.
	b, _ := json.Marshal(p)
	if string(b) != "12.50" {
		t.Fatalf("round-trip want 12.50, got %s", b)
	}
}
