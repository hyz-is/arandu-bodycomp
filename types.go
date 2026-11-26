// Package bodycomp is a "digital bioimpedance": it estimates body composition
// from self-measurements (height, weight, tape-measure circumferences and,
// optionally, caliper skinfolds) with deterministic, published formulas, and
// it compares two estimates over time.
//
// It is framework-free on purpose. It imports only the standard library, holds
// no state, does no I/O and knows nothing about WhatsApp, tenants or storage.
// The conversation that collects the measurements is a serialisable state
// machine (Session) so any channel can drive it.
//
// Every number it returns is an ESTIMATE. It is not a diagnosis, and it never
// reads a photograph: the formulas below are the only source of a percentage.
package bodycomp

import "fmt"

// Sex is the biological-sex input the formulas need. The published equations
// are fitted separately for each sex; there is no neutral coefficient set.
type Sex string

// The two values the equations are defined for.
const (
	Male   Sex = "male"
	Female Sex = "female"
)

// Valid reports whether s is one of the two defined values.
func (s Sex) Valid() bool { return s == Male || s == Female }

// Field names one measurement the package knows how to ask for, range-check
// and use. Circumferences are in centimetres, skinfolds in millimetres.
type Field string

// Circumferences (cm).
const (
	Neck       Field = "neck"
	Shoulders  Field = "shoulders"
	Chest      Field = "chest"
	Waist      Field = "waist"
	Abdomen    Field = "abdomen"
	Hip        Field = "hip"
	ArmRelaxed Field = "arm_relaxed"
	ArmFlexed  Field = "arm_flexed"
	Forearm    Field = "forearm"
	Thigh      Field = "thigh"
	Calf       Field = "calf"
)

// Skinfolds (mm), optional, for a person who owns a caliper.
const (
	FoldChest      Field = "fold_chest"
	FoldAbdomen    Field = "fold_abdomen"
	FoldThigh      Field = "fold_thigh"
	FoldTriceps    Field = "fold_triceps"
	FoldSuprailiac Field = "fold_suprailiac"
)

// Core fields of the conversation that are not circumferences.
const (
	FieldSex    Field = "sex"
	FieldAge    Field = "age"
	FieldHeight Field = "height"
	FieldWeight Field = "weight"
)

// Unit is the unit a Field is stored in.
type Unit string

// Storage units.
const (
	UnitCM Unit = "cm"
	UnitMM Unit = "mm"
	UnitKG Unit = "kg"
)

// ActivityLevel selects the multiplier that turns resting energy into daily
// energy.
type ActivityLevel string

// The five conventional levels.
const (
	Sedentary   ActivityLevel = "sedentary"
	Light       ActivityLevel = "light"
	Moderate    ActivityLevel = "moderate"
	Intense     ActivityLevel = "intense"
	VeryIntense ActivityLevel = "very_intense"
)

// Input is everything the estimate needs. Circumferences and Skinfolds are
// sparse: a missing key means "not measured", never zero.
type Input struct {
	Sex      Sex
	AgeYears int
	HeightCM float64
	WeightKG float64
	Activity ActivityLevel

	// Circumferences in centimetres, keyed by Field.
	Circumferences map[Field]float64
	// Skinfolds in millimetres, keyed by Field.
	Skinfolds map[Field]float64
}

// Method identifies one body-fat formula.
type Method string

// Body-fat methods implemented here.
const (
	MethodNavy      Method = "navy"
	MethodRFM       Method = "rfm"
	MethodCUNBAE    Method = "cun_bae"
	MethodJacksonP3 Method = "jackson_pollock_3"
)

// BodyFat is one method's estimate.
type BodyFat struct {
	Method  Method
	Percent float64
}

// Severity separates "ask again" from "ask to confirm".
type Severity string

// Issue severities.
const (
	// SeverityError means the value cannot be used: ask for it again.
	SeverityError Severity = "error"
	// SeverityWarning means the value is odd but possible: ask to confirm.
	SeverityWarning Severity = "warning"
)

// Issue is one plausibility finding, written for the person who typed the
// value (pt-BR), so a channel can show Message as it is.
type Issue struct {
	Field    Field
	Severity Severity
	Code     string
	Message  string
	// Suggestion, when set, is a corrected value (in the field's unit) the
	// person probably meant, for a one-word "sim" to accept.
	Suggestion *float64
}

// ValidationError carries the issues that stopped an estimate.
type ValidationError struct{ Issues []Issue }

// Error implements error. The text is for logs; the person reads Issues.
func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "bodycomp: invalid input"
	}
	return fmt.Sprintf("bodycomp: invalid input: %s (%s)", e.Issues[0].Code, e.Issues[0].Field)
}
