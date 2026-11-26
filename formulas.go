package bodycomp

import "math"

// This file holds the published formulas, one function each, with no
// validation and no rounding: the caller (Estimate) decides what is plausible.
// Lengths are centimetres, masses kilograms, skinfolds millimetres.

// BMI is weight over height squared (Quetelet; WHO Technical Report Series 894,
// 2000). kg/m^2.
func BMI(weightKG, heightCM float64) float64 {
	h := heightCM / 100
	return weightKG / (h * h)
}

// WaistToHeight is the waist-to-height ratio. Ashwell & Hsieh, Int J Food Sci
// Nutr 2005;56:303-307, propose 0.5 as the boundary worth attention.
func WaistToHeight(waistCM, heightCM float64) float64 { return waistCM / heightCM }

// WaistToHip is the waist-to-hip ratio. WHO, "Waist circumference and
// waist-hip ratio: report of a WHO expert consultation", Geneva 2008, marks
// abdominal obesity at >0.90 (men) and >0.85 (women).
func WaistToHip(waistCM, hipCM float64) float64 { return waistCM / hipCM }

// NavyMale is the U.S. Navy circumference method for men, metric form.
//
//	%BF = 495 / (1.0324 - 0.19077*log10(waist - neck) + 0.15456*log10(height)) - 450
//
// Hodgdon & Beckett, "Prediction of percent body fat for U.S. Navy men from
// body circumferences and height", Naval Health Research Center Report
// 84-11, 1984; body density to fat by Siri, 1961 (495/D - 450). The waist is
// taken at the navel. Requires waist > neck.
func NavyMale(waistCM, neckCM, heightCM float64) float64 {
	return 495/(1.0324-0.19077*math.Log10(waistCM-neckCM)+0.15456*math.Log10(heightCM)) - 450
}

// NavyFemale is the same method for women, which adds the hip.
//
//	%BF = 495 / (1.29579 - 0.35004*log10(waist + hip - neck) + 0.22100*log10(height)) - 450
//
// Hodgdon & Beckett, Naval Health Research Center Report 84-29, 1984. The
// waist is taken at the narrowest point.
func NavyFemale(waistCM, hipCM, neckCM, heightCM float64) float64 {
	return 495/(1.29579-0.35004*math.Log10(waistCM+hipCM-neckCM)+0.22100*math.Log10(heightCM)) - 450
}

// navyMaleInches and navyFemaleInches are the inch forms printed in OPNAVINST
// 6110.1 (the Navy's own regulation). They are kept only so a test can check
// that the metric forms above are the same equations.
func navyMaleInches(waistIn, neckIn, heightIn float64) float64 {
	return 86.010*math.Log10(waistIn-neckIn) - 70.041*math.Log10(heightIn) + 36.76
}

func navyFemaleInches(waistIn, hipIn, neckIn, heightIn float64) float64 {
	return 163.205*math.Log10(waistIn+hipIn-neckIn) - 97.684*math.Log10(heightIn) - 78.387
}

// RFM is Relative Fat Mass: 64 - 20*(height/waist) + 12*sex, with sex 0 for men
// and 1 for women. Woolcott & Bergman, Sci Rep 2018;8:10980, developed on NHANES
// 1999-2004 against DXA.
func RFM(sex Sex, heightCM, waistCM float64) float64 {
	base := 64.0
	if sex == Female {
		base = 76
	}
	return base - 20*(heightCM/waistCM)
}

// CUNBAE is the Clinica Universidad de Navarra Body Adiposity Estimator, from
// BMI, age and sex (0 men, 1 women). Gomez-Ambrosi et al., Diabetes Care
// 2012;35:383-388:
//
//	%BF = -44.988 + 0.503*age + 10.689*sex + 3.172*BMI - 0.026*BMI^2
//	      + 0.181*BMI*sex - 0.02*BMI*age - 0.005*BMI^2*sex + 0.00021*BMI^2*age
func CUNBAE(sex Sex, ageYears int, bmi float64) float64 {
	s := 0.0
	if sex == Female {
		s = 1
	}
	age := float64(ageYears)
	return -44.988 + 0.503*age + 10.689*s + 3.172*bmi - 0.026*bmi*bmi +
		0.181*bmi*s - 0.02*bmi*age - 0.005*bmi*bmi*s + 0.00021*bmi*bmi*age
}

// JacksonPollock3 is the three-site skinfold equation with Siri's conversion.
// Men use chest, abdomen and thigh; women use triceps, suprailiac and thigh.
// The skinfolds are in millimetres and summed.
//
//	men:   D = 1.10938   - 0.0008267*S + 0.0000016*S^2 - 0.0002574*age
//	women: D = 1.0994921 - 0.0009929*S + 0.0000023*S^2 - 0.0001392*age
//	%BF = 495/D - 450
//
// Jackson & Pollock, Br J Nutr 1978;40:497-504 (men); Jackson, Pollock & Ward,
// Med Sci Sports Exerc 1980;12:175-181 (women); Siri, 1961.
func JacksonPollock3(sex Sex, ageYears int, sumMM float64) float64 {
	age := float64(ageYears)
	var density float64
	if sex == Female {
		density = 1.0994921 - 0.0009929*sumMM + 0.0000023*sumMM*sumMM - 0.0001392*age
	} else {
		density = 1.10938 - 0.0008267*sumMM + 0.0000016*sumMM*sumMM - 0.0002574*age
	}
	return 495/density - 450
}

// FFMI is the fat-free mass index, lean mass (kg) over height (m) squared.
// Kouri et al., Clin J Sport Med 1995;5:223-228.
func FFMI(leanKG, heightCM float64) float64 {
	h := heightCM / 100
	return leanKG / (h * h)
}

// NormalizedFFMI adjusts FFMI to a 1.80 m reference height:
// FFMI + 6.1*(1.8 - height). Kouri et al., 1995.
func NormalizedFFMI(ffmi, heightCM float64) float64 {
	return ffmi + 6.1*(1.8-heightCM/100)
}

// MifflinStJeor is resting energy expenditure in kcal/day:
// 10*kg + 6.25*cm - 5*age + 5 (men) or - 161 (women). Mifflin et al.,
// Am J Clin Nutr 1990;51:241-247.
func MifflinStJeor(sex Sex, weightKG, heightCM float64, ageYears int) float64 {
	v := 10*weightKG + 6.25*heightCM - 5*float64(ageYears)
	if sex == Female {
		return v - 161
	}
	return v + 5
}

// KatchMcArdle is resting energy expenditure from lean mass: 370 + 21.6*LBM(kg).
// Katch & McArdle, as in McArdle, Katch & Katch, Exercise Physiology.
func KatchMcArdle(leanKG float64) float64 { return 370 + 21.6*leanKG }

// activityFactor is the conventional multiplier of resting energy (Harris and
// Benedict tradition, as tabulated in McArdle, Katch & Katch and ACSM texts).
var activityFactor = map[ActivityLevel]float64{
	Sedentary:   1.2,
	Light:       1.375,
	Moderate:    1.55,
	Intense:     1.725,
	VeryIntense: 1.9,
}

// ActivityFactor returns the multiplier for a level, or 0 for an unknown one.
func ActivityFactor(level ActivityLevel) float64 { return activityFactor[level] }

// ActivityLevels lists the levels from least to most active.
func ActivityLevels() []ActivityLevel {
	return []ActivityLevel{Sedentary, Light, Moderate, Intense, VeryIntense}
}

// ActivityLabel is the pt-BR description shown when asking for the level.
func ActivityLabel(level ActivityLevel) string {
	switch level {
	case Sedentary:
		return "Sedentário: quase sem exercício"
	case Light:
		return "Leve: treino 1 a 3 dias por semana"
	case Moderate:
		return "Moderado: treino 3 a 5 dias por semana"
	case Intense:
		return "Intenso: treino 6 a 7 dias por semana"
	case VeryIntense:
		return "Muito intenso: treino pesado todo dia ou trabalho braçal"
	}
	return ""
}

// ParseActivity reads a level from a number 1-5 or a keyword.
func ParseActivity(text string) (ActivityLevel, bool) {
	switch normalise(text) {
	case "1", "sedentario", "sedentaria", "parado":
		return Sedentary, true
	case "2", "leve", "light":
		return Light, true
	case "3", "moderado", "moderada", "medio":
		return Moderate, true
	case "4", "intenso", "intensa", "pesado":
		return Intense, true
	case "5", "muito intenso", "muito intensa", "muito pesado", "atleta":
		return VeryIntense, true
	}
	return "", false
}
