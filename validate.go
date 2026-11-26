package bodycomp

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Adults only: the equations are fitted on adults, and a body-composition
// estimate for a minor is sensitive data that needs a different legal basis
// (LGPD art. 14, ECA Digital). The conversation stops at the age question.
const (
	MinAgeYears = 18
	MaxAgeYears = 90
)

// br formats a number the way a Brazilian reads it: comma, no trailing zeros.
func br(v float64) string {
	s := strconv.FormatFloat(math.Round(v*10)/10, 'f', 1, 64)
	s = strings.TrimSuffix(s, ".0")
	return strings.Replace(s, ".", ",", 1)
}

func unitSuffix(u Unit) string { return " " + string(u) }

// ValidateAge checks the age answer.
func ValidateAge(years int) []Issue {
	switch {
	case years <= 0 || years > MaxAgeYears:
		return []Issue{{Field: FieldAge, Severity: SeverityError, Code: "age_range",
			Message: fmt.Sprintf("%d anos não parece certo. Confira a idade (ou mande a data de nascimento, como 15/03/1990).", years)}}
	case years < MinAgeYears:
		return []Issue{{Field: FieldAge, Severity: SeverityError, Code: "minor",
			Message: "A bioimpedância digital é só para maiores de 18 anos. Para menores, procure um profissional de saúde."}}
	}
	return nil
}

// ValidateValue checks one number against the field's humanly possible range.
// Out of range is an error; a value that becomes plausible after a likely slip
// (a lost decimal point, pounds, inches) carries that Suggestion.
func ValidateValue(f Field, v float64) []Issue { return validateValue(f, v, nil) }

// validateValue is ValidateValue with a filter on suggestions: a correction the
// caller finds implausible (it would break a relation with another answer) is
// not offered.
func validateValue(f Field, v float64, accept func(float64) bool) []Issue {
	spec, ok := SpecFor(f)
	if !ok {
		return nil
	}
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return []Issue{{Field: f, Severity: SeverityError, Code: "not_positive",
			Message: fmt.Sprintf("Preciso de um número maior que zero para %s.", spec.Label)}}
	}
	if v >= spec.Min && v <= spec.Max {
		return nil
	}
	issue := Issue{Field: f, Severity: SeverityError, Code: "range"}
	for _, candidate := range slips(spec, v) {
		if candidate >= spec.Min && candidate <= spec.Max && (accept == nil || accept(candidate)) {
			c := math.Round(candidate*10) / 10
			issue.Suggestion = &c
			issue.Message = fmt.Sprintf("%s de %s%s não parece certo. Você quis dizer %s%s? Responda \"sim\" ou mande o valor certo.",
				capitalise(spec.Label), br(v), unitSuffix(spec.Unit), br(c), unitSuffix(spec.Unit))
			return []Issue{issue}
		}
	}
	issue.Message = fmt.Sprintf("%s de %s%s está fora do que é possível (entre %s e %s%s). Meça de novo e mande o valor.",
		capitalise(spec.Label), br(v), unitSuffix(spec.Unit), br(spec.Min), br(spec.Max), unitSuffix(spec.Unit))
	return []Issue{issue}
}

// slips lists corrections for the mistakes people really make, in the order
// they are tried: a decimal point swallowed or added, then a different unit.
func slips(spec Spec, v float64) []float64 {
	out := []float64{v / 10, v * 10}
	switch spec.Unit {
	case UnitKG:
		out = append(out, v*0.45359237) // pounds without the unit
	case UnitCM:
		out = append(out, v*2.54) // inches without the unit
	}
	return out
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// Validate checks a whole Input: each value against its range, then the
// relations between values that a mistyped number breaks (a waist smaller
// than the neck, a flexed arm smaller than the relaxed one). It returns every
// finding, errors first, ordered by field so the output is stable.
func Validate(in Input) []Issue {
	var issues []Issue
	if !in.Sex.Valid() {
		issues = append(issues, Issue{Field: FieldSex, Severity: SeverityError, Code: "sex",
			Message: "Preciso saber se as fórmulas são as masculinas ou as femininas."})
	}
	issues = append(issues, ValidateAge(in.AgeYears)...)
	issues = append(issues, ValidateValue(FieldHeight, in.HeightCM)...)
	issues = append(issues, ValidateValue(FieldWeight, in.WeightKG)...)
	if _, ok := in.Circumferences[Waist]; !ok {
		issues = append(issues, Issue{Field: Waist, Severity: SeverityError, Code: "missing",
			Message: "Falta a cintura: é a medida mais importante da estimativa."})
	}
	for f, v := range in.Circumferences {
		issues = append(issues, ValidateValue(f, v)...)
	}
	for f, v := range in.Skinfolds {
		issues = append(issues, ValidateValue(f, v)...)
	}
	issues = append(issues, crossChecks(in)...)

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity == SeverityError
		}
		return issues[i].Field < issues[j].Field
	})
	return issues
}

// HasErrors reports whether any issue stops the estimate.
func HasErrors(issues []Issue) bool {
	for _, i := range issues {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}

// CheckRelation validates one new circumference against the values already
// answered, so a conversation can refuse a number the moment it is typed
// instead of at the end. have holds the other circumferences in cm.
func CheckRelation(f Field, v float64, heightCM float64, have map[Field]float64) []Issue {
	merged := make(map[Field]float64, len(have)+1)
	for k, x := range have {
		merged[k] = x
	}
	merged[f] = v
	var out []Issue
	for _, i := range crossChecks(Input{HeightCM: heightCM, Circumferences: merged}) {
		if i.Field == f {
			out = append(out, i)
		}
	}
	return out
}

func crossChecks(in Input) []Issue {
	var out []Issue
	c := in.Circumferences
	waist, hasWaist := c[Waist]
	neck, hasNeck := c[Neck]
	hip, hasHip := c[Hip]
	relaxed, hasRelaxed := c[ArmRelaxed]
	flexed, hasFlexed := c[ArmFlexed]

	if hasWaist && hasNeck && waist <= neck {
		out = append(out, Issue{Field: Waist, Severity: SeverityError, Code: "waist_le_neck",
			Message: fmt.Sprintf("A cintura (%s cm) ficou menor que o pescoço (%s cm), o que não acontece. Meça a cintura de novo, na altura do umbigo.", br(waist), br(neck))})
	}
	if hasWaist && in.HeightCM > 0 {
		ratio := waist / in.HeightCM
		switch {
		case ratio >= 1:
			out = append(out, Issue{Field: Waist, Severity: SeverityError, Code: "waist_ge_height",
				Message: fmt.Sprintf("Cintura de %s cm para %s cm de altura é improvável. Confira as duas medidas.", br(waist), br(in.HeightCM))})
		case ratio < 0.3:
			out = append(out, Issue{Field: Waist, Severity: SeverityError, Code: "waist_tiny",
				Message: fmt.Sprintf("Cintura de %s cm para %s cm de altura é pequena demais. Confira se a fita estava no lugar certo.", br(waist), br(in.HeightCM))})
		}
	}
	if hasWaist && hasHip {
		if r := hip / waist; r < 0.6 || r > 1.8 {
			out = append(out, Issue{Field: Hip, Severity: SeverityWarning, Code: "hip_waist_odd",
				Message: fmt.Sprintf("Quadril de %s cm e cintura de %s cm formam uma proporção incomum. Se estiver certo, responda \"sim\"; se não, mande o valor certo.", br(hip), br(waist))})
		}
	}
	if hasRelaxed && hasFlexed && flexed < relaxed {
		out = append(out, Issue{Field: ArmFlexed, Severity: SeverityError, Code: "flexed_lt_relaxed",
			Message: fmt.Sprintf("O braço contraído (%s cm) ficou menor que o relaxado (%s cm). Contraia o bíceps com força e meça de novo.", br(flexed), br(relaxed))})
	}
	if forearm, ok := c[Forearm]; ok && hasRelaxed && forearm > relaxed {
		out = append(out, Issue{Field: Forearm, Severity: SeverityWarning, Code: "forearm_gt_arm",
			Message: "O antebraço ficou maior que o braço relaxado, o que é raro. Se estiver certo, responda \"sim\"; se não, mande o valor certo."})
	}
	if calf, ok := c[Calf]; ok {
		if thigh, ok := c[Thigh]; ok && calf > thigh {
			out = append(out, Issue{Field: Calf, Severity: SeverityWarning, Code: "calf_gt_thigh",
				Message: "A panturrilha ficou maior que a coxa, o que é raro. Se estiver certo, responda \"sim\"; se não, mande o valor certo."})
		}
	}
	if in.HeightCM > 0 && in.WeightKG > 0 {
		if b := BMI(in.WeightKG, in.HeightCM); b < 12 || b > 70 {
			out = append(out, Issue{Field: FieldWeight, Severity: SeverityError, Code: "bmi_implausible",
				Message: fmt.Sprintf("Peso de %s kg com %s cm de altura dá um IMC de %s, improvável. Confira o peso e a altura.", br(in.WeightKG), br(in.HeightCM), br(b))})
		}
	}
	return out
}
