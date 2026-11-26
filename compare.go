package bodycomp

import (
	"fmt"
	"math"
)

// FatNoisePP is the body-fat change, in percentage points, below which two
// estimates are not distinguishable: the formulas' own error is larger.
const FatNoisePP = 1.5

// Delta is the change between two results. Positive means the second result
// is higher. Circumference changes only cover sites present in both.
type Delta struct {
	Days           int
	WeightKG       float64
	BMI            float64
	BodyFatPP      float64
	FatMassKG      float64
	LeanMassKG     float64
	FFMI           float64
	WaistCM        float64
	Circumferences map[Field]float64
	// FatWithinNoise is true when the body-fat change is smaller than FatNoisePP.
	FatWithinNoise bool
}

// Compare returns the change from prev to cur; days is the time between them.
func Compare(prev, cur Result, days int) Delta {
	d := Delta{
		Days:           days,
		WeightKG:       round1(cur.Input.WeightKG - prev.Input.WeightKG),
		BMI:            round1(cur.BMI - prev.BMI),
		BodyFatPP:      round1(cur.BodyFatPercent - prev.BodyFatPercent),
		FatMassKG:      round1(cur.FatMassKG - prev.FatMassKG),
		LeanMassKG:     round1(cur.LeanMassKG - prev.LeanMassKG),
		FFMI:           round1(cur.FFMI - prev.FFMI),
		Circumferences: map[Field]float64{},
	}
	d.FatWithinNoise = math.Abs(d.BodyFatPP) < FatNoisePP
	for f, v := range cur.Input.Circumferences {
		if p, ok := prev.Input.Circumferences[f]; ok {
			d.Circumferences[f] = round1(v - p)
		}
	}
	d.WaistCM = d.Circumferences[Waist]
	return d
}

// Summary writes the change as short pt-BR lines, in the order a person cares
// about: body fat, lean mass, weight, waist. It states a body-fat change inside
// the formulas' noise as "sem mudança clara" and never promises a cause.
func (d Delta) Summary() []string {
	var lines []string
	switch {
	case d.FatWithinNoise:
		lines = append(lines, fmt.Sprintf("Gordura corporal: %s pontos, sem mudança clara (dentro da margem de erro das fórmulas).", signed(d.BodyFatPP)))
	default:
		lines = append(lines, fmt.Sprintf("Gordura corporal: %s pontos percentuais.", signed(d.BodyFatPP)))
	}
	lines = append(lines,
		fmt.Sprintf("Massa magra: %s kg. Massa gorda: %s kg.", signed(d.LeanMassKG), signed(d.FatMassKG)),
		fmt.Sprintf("Peso: %s kg.", signed(d.WeightKG)),
	)
	if _, ok := d.Circumferences[Waist]; ok {
		lines = append(lines, fmt.Sprintf("Cintura: %s cm.", signed(d.WaistCM)))
	}
	return lines
}

func signed(v float64) string {
	s := br(math.Abs(v))
	switch {
	case v > 0:
		return "+" + s
	case v < 0:
		return "-" + s
	}
	return s
}
