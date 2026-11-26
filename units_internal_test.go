package bodycomp

import (
	"errors"
	"testing"
	"time"
)

func TestParseLengthCM(t *testing.T) {
	ok := map[string]float64{
		"175": 175, "175cm": 175, "175 cm": 175, "175 CM": 175, "175 cms": 175, "175 centímetros": 175,
		"1,75": 175, "1.75": 175, "1,75m": 175, "1.75 m": 175, "1,75 metros": 175, "1m75": 175, "1 m 75": 175,
		"69 in": 175.26, "69 polegadas": 175.26, `69"`: 175.26, "5'9": 175.26, `5'9"`: 175.26, "5 ft 9 in": 175.26,
		"85,5": 85.5, "85.5cm": 85.5, "8O": 80, "1l0": 110, "850 mm": 85, "  92  ": 92, "1.234,5 mm": 123.45,
	}
	for in, want := range ok {
		got, err := ParseLengthCM(in)
		if err != nil || got != want {
			t.Errorf("ParseLengthCM(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "cm", "1,2,3", "1..2", "kg"} {
		if _, err := ParseLengthCM(in); err == nil {
			t.Errorf("ParseLengthCM(%q) must fail", in)
		}
	}
	if _, err := ParseLengthCM("70 kg"); !errors.Is(err, ErrWrongUnit) {
		t.Errorf("a weight unit for a length must be ErrWrongUnit, got %v", err)
	}
}

func TestParseMassKG(t *testing.T) {
	ok := map[string]float64{
		"72": 72, "72,5": 72.5, "72.5kg": 72.5, "72 quilos": 72, "72 kgs": 72, "72 Kilos": 72,
		"154 lb": 69.85, "154lbs": 69.85, "154 libras": 69.85, "7O": 70, "72500 g": 72.5,
	}
	for in, want := range ok {
		got, err := ParseMassKG(in)
		if err != nil || got != want {
			t.Errorf("ParseMassKG(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseMassKG("175 cm"); !errors.Is(err, ErrWrongUnit) {
		t.Errorf("a length unit for a weight must be ErrWrongUnit, got %v", err)
	}
	if _, err := ParseMassKG("quilos"); err == nil {
		t.Error("a unit with no number must fail")
	}
}

func TestParseSkinfoldMM(t *testing.T) {
	for in, want := range map[string]float64{"12": 12, "12 mm": 12, "1,2 cm": 12, "11,5": 11.5} {
		if got, err := ParseSkinfoldMM(in); err != nil || got != want {
			t.Errorf("ParseSkinfoldMM(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
}

func TestParseAge(t *testing.T) {
	now := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	cases := map[string]int{
		"34": 34, "34 anos": 34, "1 ano": 1,
		"15/03/1990": 36, "25/11/1990": 36, "26/11/1990": 35, "15-03-1990": 36, "15.03.90": 36, "01/01/05": 21,
	}
	for in, want := range cases {
		got, err := ParseAge(in, now)
		if err != nil || got != want {
			t.Errorf("ParseAge(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseAge("31/02/1990", now); err == nil {
		t.Error("an impossible date must fail")
	}
	if _, err := ParseAge("01/01/2030", now); !errors.Is(err, ErrFutureBirth) {
		t.Errorf("a future date must be ErrFutureBirth, got %v", err)
	}
	if _, err := ParseAge("trinta", now); err == nil {
		t.Error("words must fail")
	}
}

func TestParseSex(t *testing.T) {
	for in, want := range map[string]Sex{"homem": Male, "Masculino": Male, "M": Male, "mulher": Female, "Feminino.": Female, "f": Female} {
		if got, err := ParseSex(in); err != nil || got != want {
			t.Errorf("ParseSex(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := ParseSex("talvez"); err == nil {
		t.Error("unknown word must fail")
	}
}
