package bodycomp

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Errors returned by the parsers. The conversation turns each into a short
// question; the text here is for logs.
var (
	ErrNoNumber    = errors.New("bodycomp: no number in the text")
	ErrBadNumber   = errors.New("bodycomp: the number is not readable")
	ErrWrongUnit   = errors.New("bodycomp: that unit does not fit this measurement")
	ErrNotASex     = errors.New("bodycomp: not a sex")
	ErrNotAnAge    = errors.New("bodycomp: not an age")
	ErrFutureBirth = errors.New("bodycomp: the birth date is in the future")
)

type dimension int

const (
	dimLength dimension = iota
	dimMass
)

// unitDef maps a spelled unit (lower case, no accents) to its size in the
// base unit of its dimension: centimetres for length, kilograms for mass.
type unitDef struct {
	word   string
	dim    dimension
	factor float64
}

// units is ordered longest first so "centimetros" is tried before "cm" and
// "lbs" before "lb" before the bare "m" or "g" that would otherwise swallow a
// suffix. Typos people actually send ("cms", "kgs", "quilos", "mts") are listed
// as units, which is the whole of the typo correction for units.
var units = []unitDef{
	{"centimetros", dimLength, 1}, {"centimetro", dimLength, 1}, {"milimetros", dimLength, 0.1},
	{"milimetro", dimLength, 0.1}, {"polegadas", dimLength, 2.54}, {"polegada", dimLength, 2.54},
	{"quilogramas", dimMass, 1}, {"quilograma", dimMass, 1}, {"kilogramas", dimMass, 1},
	{"metros", dimLength, 100}, {"metro", dimLength, 100}, {"inches", dimLength, 2.54},
	{"quilos", dimMass, 1}, {"kilos", dimMass, 1}, {"libras", dimMass, 0.45359237},
	{"inch", dimLength, 2.54}, {"quilo", dimMass, 1}, {"kilo", dimMass, 1}, {"libra", dimMass, 0.45359237},
	{"gramas", dimMass, 0.001}, {"grama", dimMass, 0.001},
	{"cms", dimLength, 1}, {"mts", dimLength, 100}, {"kgs", dimMass, 1}, {"lbs", dimMass, 0.45359237},
	{"pol", dimLength, 2.54},
	{"cm", dimLength, 1}, {"mm", dimLength, 0.1}, {"kg", dimMass, 1}, {"lb", dimMass, 0.45359237},
	{"in", dimLength, 2.54}, {"mt", dimLength, 100}, {`"`, dimLength, 2.54},
	{"m", dimLength, 100}, {"g", dimMass, 0.001},
}

var (
	feetInches = regexp.MustCompile(`^(\d+)\s*(?:'|ft|pes|pe)\s*(\d+(?:[.,]\d+)?)?\s*(?:"|in|pol)?$`)
	metreCm    = regexp.MustCompile(`^(\d)\s*m\s*(\d{1,2})$`)
	// digitLike is what a number is allowed to be made of before typo repair:
	// digits, separators, and the letters a thumb types for 0 and 1.
	digitLike = regexp.MustCompile(`^[0-9.,oOlI]+$`)
)

var accentless = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o",
	"ô", "o", "õ", "o", "ú", "u", "ç", "c",
	"’", "'", "′", "'", "´", "'", "”", `"`, "“", `"`, "″", `"`, "''", `"`,
)

func normalise(text string) string {
	return strings.TrimSpace(accentless.Replace(strings.ToLower(strings.TrimSpace(text))))
}

// splitUnit separates "70,5 kg" into ("70,5", unit). A text with no known unit
// returns a nil unit and the whole text.
func splitUnit(s string) (string, *unitDef) {
	trimmed := strings.TrimRight(s, ". ")
	for i := range units {
		u := &units[i]
		if !strings.HasSuffix(trimmed, u.word) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimSuffix(trimmed, u.word))
		if rest != "" {
			return rest, u
		}
	}
	return s, nil
}

// parseNumber reads a decimal with comma or point, repairing the usual thumb
// slips: a letter O or l typed for 0 or 1 inside a number ("7O" -> 70), and
// stray spaces. A single separator is always the decimal one; two different
// separators mean the last is decimal ("1.234,5"); two equal separators are a
// typo and are refused.
func parseNumber(s string) (float64, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if s == "" {
		return 0, ErrNoNumber
	}
	if !digitLike.MatchString(s) {
		return 0, ErrBadNumber
	}
	hasDigit := false
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
			b.WriteRune(r)
		case r == 'o' || r == 'O':
			b.WriteRune('0')
		case r == 'l' || r == 'I':
			b.WriteRune('1')
		default:
			b.WriteRune(r)
		}
	}
	if !hasDigit {
		return 0, ErrBadNumber
	}
	s = b.String()
	commas, points := strings.Count(s, ","), strings.Count(s, ".")
	switch {
	case commas+points == 0:
	case commas > 0 && points > 0:
		last := strings.LastIndexAny(s, ",.")
		s = strings.NewReplacer(",", "", ".", "").Replace(s[:last]) + "." + s[last+1:]
	case commas > 1 || points > 1:
		return 0, ErrBadNumber
	default:
		s = strings.Replace(s, ",", ".", 1)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0, ErrBadNumber
	}
	return v, nil
}

// ParseLengthCM reads a length typed by a person and returns centimetres.
// It understands "175", "175 cm", "1,75", "1.75 m", "1m75", "69 in", "5'9",
// "5 ft 9 in" and common misspellings of the units. A bare number under 3 is
// metres (nobody is 2 cm tall), any other bare number is centimetres.
func ParseLengthCM(text string) (float64, error) {
	s := normalise(text)
	if s == "" {
		return 0, ErrNoNumber
	}
	if m := feetInches.FindStringSubmatch(s); m != nil {
		feet, _ := strconv.ParseFloat(m[1], 64)
		inches := 0.0
		if m[2] != "" {
			v, err := parseNumber(m[2])
			if err != nil {
				return 0, err
			}
			inches = v
		}
		return round2(feet*30.48 + inches*2.54), nil
	}
	if m := metreCm.FindStringSubmatch(s); m != nil {
		metres, _ := strconv.ParseFloat(m[1], 64)
		cm, _ := strconv.ParseFloat(m[2], 64)
		if len(m[2]) == 1 {
			cm *= 10
		}
		return round2(metres*100 + cm), nil
	}
	number, unit := splitUnit(s)
	v, err := parseNumber(number)
	if err != nil {
		return 0, err
	}
	if unit == nil {
		if v < 3 {
			return round2(v * 100), nil
		}
		return round2(v), nil
	}
	if unit.dim != dimLength {
		return 0, ErrWrongUnit
	}
	return round2(v * unit.factor), nil
}

// ParseMassKG reads a weight and returns kilograms. A bare number is
// kilograms; "154 lb" and "154 libras" are pounds.
func ParseMassKG(text string) (float64, error) {
	s := normalise(text)
	if s == "" {
		return 0, ErrNoNumber
	}
	number, unit := splitUnit(s)
	v, err := parseNumber(number)
	if err != nil {
		return 0, err
	}
	if unit == nil {
		return round2(v), nil
	}
	if unit.dim != dimMass {
		return 0, ErrWrongUnit
	}
	return round2(v * unit.factor), nil
}

// ParseSkinfoldMM reads a skinfold and returns millimetres. A bare number is
// millimetres; "1,2 cm" is 12 mm.
func ParseSkinfoldMM(text string) (float64, error) {
	s := normalise(text)
	if s == "" {
		return 0, ErrNoNumber
	}
	number, unit := splitUnit(s)
	v, err := parseNumber(number)
	if err != nil {
		return 0, err
	}
	if unit == nil {
		return round2(v), nil
	}
	if unit.dim != dimLength {
		return 0, ErrWrongUnit
	}
	return round2(v * unit.factor * 10), nil
}

var (
	birthDate = regexp.MustCompile(`^(\d{1,2})[/.\-](\d{1,2})[/.\-](\d{4}|\d{2})$`)
	ageWords  = regexp.MustCompile(`^(\d{1,3})\s*(?:anos?|a)?$`)
)

// ParseAge reads an age in years from "34", "34 anos" or a birth date
// ("15/03/1990"), computed at now. A two-digit year is read as 19xx when that
// would make the person at most 100, otherwise as 20xx.
func ParseAge(text string, now time.Time) (int, error) {
	s := normalise(text)
	if m := birthDate.FindStringSubmatch(s); m != nil {
		d, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		y, _ := strconv.Atoi(m[3])
		if len(m[3]) == 2 {
			y += 1900
			if now.Year()-y > 100 {
				y += 100
			}
		}
		born := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
		if born.Day() != d || int(born.Month()) != mo {
			return 0, ErrNotAnAge
		}
		if born.After(now) {
			return 0, ErrFutureBirth
		}
		return AgeAt(born, now), nil
	}
	if m := ageWords.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, nil
	}
	return 0, ErrNotAnAge
}

// AgeAt returns the completed years between born and now.
func AgeAt(born, now time.Time) int {
	years := now.Year() - born.Year()
	if now.Month() < born.Month() || (now.Month() == born.Month() && now.Day() < born.Day()) {
		years--
	}
	return years
}

// ParseSex reads a sex from the words people use in a chat.
func ParseSex(text string) (Sex, error) {
	switch strings.Trim(normalise(text), " .!") {
	case "m", "h", "homem", "masculino", "masc", "male", "macho", "cara", "menino":
		return Male, nil
	case "f", "mulher", "feminino", "fem", "female", "femea", "menina":
		return Female, nil
	}
	return "", ErrNotASex
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
