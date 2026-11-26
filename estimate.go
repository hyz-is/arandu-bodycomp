package bodycomp

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Disclaimer must accompany every result a person sees.
const Disclaimer = "Isto é uma estimativa feita com fórmulas publicadas a partir das suas medidas. " +
	"Não é diagnóstico e não substitui a avaliação de um profissional de saúde."

// Agreement says how closely the available methods agree.
type Agreement string

// Agreement levels. Spread is the gap, in percentage points, between the
// highest and the lowest method.
const (
	AgreementSingle Agreement = "single" // only one method could run
	AgreementHigh   Agreement = "high"   // spread up to 3 points
	AgreementMedium Agreement = "medium" // spread up to 6 points
	AgreementLow    Agreement = "low"    // spread above 6 points
)

// Plausible window for any single method's output. Outside it the method is
// dropped with a warning instead of being averaged in.
const (
	minPlausibleFat = 2.0
	maxPlausibleFat = 65.0
)

// LevelEnergy is daily energy at one activity level.
type LevelEnergy struct {
	Level  ActivityLevel `json:"level"`
	Factor float64       `json:"factor"`
	// Mifflin is BMR (Mifflin-St Jeor) times Factor; Katch uses Katch-McArdle.
	Mifflin float64 `json:"mifflin_kcal"`
	Katch   float64 `json:"katch_kcal"`
}

// Energy groups resting and daily energy, in kcal/day.
type Energy struct {
	BMRMifflin float64       `json:"bmr_mifflin_kcal"`
	BMRKatch   float64       `json:"bmr_katch_kcal"`
	Activity   ActivityLevel `json:"activity,omitempty"`
	// TDEE is Mifflin times the chosen activity factor; zero when no level
	// was given. Levels always lists every level.
	TDEE   float64       `json:"tdee_kcal,omitempty"`
	Levels []LevelEnergy `json:"levels"`
}

// Result is one estimate. It marshals to JSON so a store can keep the whole
// thing beside the input that produced it.
type Result struct {
	Input Input `json:"input"`

	BMI      float64 `json:"bmi"`
	BMIClass string  `json:"bmi_class"`

	WaistToHeight     *float64 `json:"waist_to_height,omitempty"`
	WaistToHeightNote string   `json:"waist_to_height_note,omitempty"`
	WaistToHip        *float64 `json:"waist_to_hip,omitempty"`
	WaistToHipNote    string   `json:"waist_to_hip_note,omitempty"`

	// Methods are the per-method body-fat estimates that ran.
	Methods []BodyFat `json:"methods"`
	// BodyFatPercent is the median of Methods (the mean of two).
	BodyFatPercent float64   `json:"body_fat_percent"`
	SpreadPP       float64   `json:"spread_pp"`
	Agreement      Agreement `json:"agreement"`
	// FatCategory places BodyFatPercent in the ACE reference bands.
	FatCategory string `json:"fat_category"`

	FatMassKG      float64 `json:"fat_mass_kg"`
	LeanMassKG     float64 `json:"lean_mass_kg"`
	FFMI           float64 `json:"ffmi"`
	NormalizedFFMI float64 `json:"normalized_ffmi"`

	Energy Energy `json:"energy"`

	// Warnings are plain-language notes (a dropped method, a confirmed odd
	// value). Disclaimer is always set.
	Warnings   []string `json:"warnings,omitempty"`
	Disclaimer string   `json:"disclaimer"`
}

// Estimate validates the input and computes every metric the data allows.
//
// It returns a *ValidationError when an error-level issue is present: the
// caller asks again instead of showing a number. Warning-level issues do not
// stop it; they come back in Result.Warnings.
func Estimate(in Input) (Result, error) {
	issues := Validate(in)
	if HasErrors(issues) {
		var errs []Issue
		for _, i := range issues {
			if i.Severity == SeverityError {
				errs = append(errs, i)
			}
		}
		return Result{}, &ValidationError{Issues: errs}
	}

	r := Result{Input: in, Disclaimer: Disclaimer}
	for _, i := range issues {
		r.Warnings = append(r.Warnings, i.Message)
	}

	r.BMI = round1(BMI(in.WeightKG, in.HeightCM))
	r.BMIClass = BMIClass(r.BMI)

	waist := in.Circumferences[Waist]
	wh := round2(WaistToHeight(waist, in.HeightCM))
	r.WaistToHeight = &wh
	r.WaistToHeightNote = waistToHeightNote(wh)
	if hip, ok := in.Circumferences[Hip]; ok {
		whr := round2(WaistToHip(waist, hip))
		r.WaistToHip = &whr
		r.WaistToHipNote = waistToHipNote(in.Sex, whr)
	}

	r.Methods, r.Warnings = bodyFatMethods(in, r.Warnings)
	if len(r.Methods) == 0 {
		return Result{}, &ValidationError{Issues: []Issue{{
			Field: Waist, Severity: SeverityError, Code: "no_estimate",
			Message: "Com essas medidas não cheguei a um resultado possível. Confira a cintura e o pescoço e tente de novo.",
		}}}
	}

	r.BodyFatPercent, r.SpreadPP, r.Agreement = combine(r.Methods)
	r.FatCategory = FatCategory(in.Sex, r.BodyFatPercent)
	r.FatMassKG = round1(in.WeightKG * r.BodyFatPercent / 100)
	r.LeanMassKG = round1(in.WeightKG - r.FatMassKG)
	r.FFMI = round1(FFMI(r.LeanMassKG, in.HeightCM))
	r.NormalizedFFMI = round1(NormalizedFFMI(r.FFMI, in.HeightCM))
	r.Energy = energy(in, r.LeanMassKG)
	return r, nil
}

func bodyFatMethods(in Input, warnings []string) ([]BodyFat, []string) {
	var out []BodyFat
	add := func(m Method, v float64) {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < minPlausibleFat || v > maxPlausibleFat {
			warnings = append(warnings, fmt.Sprintf("O método %s deu %s%%, o que é improvável, então não entrou na média. Confira as medidas usadas por ele.",
				MethodName(m), br(v)))
			return
		}
		out = append(out, BodyFat{Method: m, Percent: round1(v)})
	}
	c := in.Circumferences
	waist := c[Waist]
	neck, hasNeck := c[Neck]
	hip, hasHip := c[Hip]

	switch {
	case in.Sex == Male && hasNeck:
		add(MethodNavy, NavyMale(waist, neck, in.HeightCM))
	case in.Sex == Female && hasNeck && hasHip:
		add(MethodNavy, NavyFemale(waist, hip, neck, in.HeightCM))
	}
	add(MethodRFM, RFM(in.Sex, in.HeightCM, waist))
	add(MethodCUNBAE, CUNBAE(in.Sex, in.AgeYears, BMI(in.WeightKG, in.HeightCM)))

	if sum, ok := foldSum(in); ok {
		add(MethodJacksonP3, JacksonPollock3(in.Sex, in.AgeYears, sum))
	}
	return out, warnings
}

// foldSum sums the three Jackson-Pollock sites; false when any is missing.
func foldSum(in Input) (float64, bool) {
	sum := 0.0
	for _, f := range FoldsFor(in.Sex) {
		v, ok := in.Skinfolds[f]
		if !ok {
			return 0, false
		}
		sum += v
	}
	return sum, true
}

// combine returns the median body fat, the spread and the agreement level.
func combine(methods []BodyFat) (float64, float64, Agreement) {
	vals := make([]float64, len(methods))
	for i, m := range methods {
		vals[i] = m.Percent
	}
	sort.Float64s(vals)
	n := len(vals)
	var mid float64
	if n%2 == 1 {
		mid = vals[n/2]
	} else {
		mid = (vals[n/2-1] + vals[n/2]) / 2
	}
	spread := round1(vals[n-1] - vals[0])
	level := AgreementSingle
	switch {
	case n == 1:
	case spread <= 3:
		level = AgreementHigh
	case spread <= 6:
		level = AgreementMedium
	default:
		level = AgreementLow
	}
	return round1(mid), spread, level
}

func energy(in Input, leanKG float64) Energy {
	bmr := MifflinStJeor(in.Sex, in.WeightKG, in.HeightCM, in.AgeYears)
	katch := KatchMcArdle(leanKG)
	e := Energy{BMRMifflin: math.Round(bmr), BMRKatch: math.Round(katch), Activity: in.Activity}
	for _, l := range ActivityLevels() {
		f := ActivityFactor(l)
		e.Levels = append(e.Levels, LevelEnergy{Level: l, Factor: f, Mifflin: math.Round(bmr * f), Katch: math.Round(katch * f)})
	}
	if f := ActivityFactor(in.Activity); f > 0 {
		e.TDEE = math.Round(bmr * f)
	}
	return e
}

// BMIClass labels a BMI with the WHO adult bands (WHO TRS 894, 2000), worded
// as a reference band and not as a condition.
func BMIClass(bmi float64) string {
	switch {
	case bmi < 18.5:
		return "abaixo da faixa de referência (menos de 18,5)"
	case bmi < 25:
		return "dentro da faixa de referência (18,5 a 24,9)"
	case bmi < 30:
		return "acima da faixa de referência: sobrepeso (25 a 29,9)"
	case bmi < 35:
		return "obesidade grau I (30 a 34,9)"
	case bmi < 40:
		return "obesidade grau II (35 a 39,9)"
	}
	return "obesidade grau III (40 ou mais)"
}

func waistToHeightNote(r float64) string {
	switch {
	case r < 0.5:
		return "abaixo de 0,5, a referência mais usada"
	case r < 0.6:
		return "acima de 0,5: vale acompanhar a cintura"
	}
	return "bem acima de 0,5: vale acompanhar a cintura com atenção"
}

func waistToHipNote(sex Sex, r float64) string {
	limit := 0.90
	if sex == Female {
		limit = 0.85
	}
	if r > limit {
		return fmt.Sprintf("acima de %s, o limite de referência da OMS para %s", br2(limit), sexWord(sex))
	}
	return fmt.Sprintf("até %s, dentro da referência da OMS para %s", br2(limit), sexWord(sex))
}

func sexWord(s Sex) string {
	if s == Female {
		return "mulheres"
	}
	return "homens"
}

func br2(v float64) string { return strings.Replace(fmt.Sprintf("%.2f", v), ".", ",", 1) }

// FatCategory places a body-fat percentage in the American Council on
// Exercise reference bands (essential, athletes, fitness, average, above).
// These are descriptive bands for fitness, not health thresholds.
func FatCategory(sex Sex, pct float64) string {
	type band struct {
		upTo  float64
		label string
	}
	var bands []band
	if sex == Female {
		bands = []band{{10, "abaixo da gordura essencial"}, {14, "gordura essencial"}, {21, "faixa de atleta"}, {25, "faixa fitness"}, {32, "faixa média"}}
	} else {
		bands = []band{{2, "abaixo da gordura essencial"}, {6, "gordura essencial"}, {14, "faixa de atleta"}, {18, "faixa fitness"}, {25, "faixa média"}}
	}
	for _, b := range bands {
		if pct < b.upTo {
			return b.label
		}
	}
	return "acima da faixa média"
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
