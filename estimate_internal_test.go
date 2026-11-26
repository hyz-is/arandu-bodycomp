package bodycomp

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func maleInput() Input {
	return Input{
		Sex: Male, AgeYears: 30, HeightCM: 180, WeightKG: 80, Activity: Moderate,
		Circumferences: map[Field]float64{Neck: 40, Waist: 90, Hip: 100},
	}
}

func femaleInput() Input {
	return Input{
		Sex: Female, AgeYears: 30, HeightCM: 165, WeightKG: 60, Activity: Light,
		Circumferences: map[Field]float64{Neck: 34, Waist: 75, Hip: 100},
	}
}

func method(r Result, m Method) (float64, bool) {
	for _, b := range r.Methods {
		if b.Method == m {
			return b.Percent, true
		}
	}
	return 0, false
}

func TestEstimateMale(t *testing.T) {
	r, err := Estimate(maleInput())
	if err != nil {
		t.Fatal(err)
	}
	near(t, "BMI", r.BMI, 24.7, 1e-9)
	if navy, ok := method(r, MethodNavy); !ok || navy != 18.4 {
		t.Fatalf("navy = %v %v, want 18.4", navy, ok)
	}
	// RFM: 64 - 20*(180/90) = 24.0 ; CUN-BAE with BMI 24.69 age 30.
	if rfm, _ := method(r, MethodRFM); rfm != 24 {
		t.Fatalf("rfm = %v, want 24", rfm)
	}
	if _, ok := method(r, MethodCUNBAE); !ok {
		t.Fatal("cun-bae missing")
	}
	if _, ok := method(r, MethodJacksonP3); ok {
		t.Fatal("jackson-pollock needs skinfolds and must not run without them")
	}
	// The consensus is the median of three, so it equals one of them.
	if _, found := func() (float64, bool) {
		for _, m := range r.Methods {
			if m.Percent == r.BodyFatPercent {
				return m.Percent, true
			}
		}
		return 0, false
	}(); !found {
		t.Fatalf("median %v is not one of %v", r.BodyFatPercent, r.Methods)
	}
	near(t, "fat+lean = weight", r.FatMassKG+r.LeanMassKG, 80, 0.11)
	if r.Energy.BMRMifflin != 1780 {
		t.Fatalf("BMR = %v, want 1780", r.Energy.BMRMifflin)
	}
	if r.Energy.TDEE != 2759 { // 1780 * 1.55 = 2759
		t.Fatalf("TDEE = %v, want 2759", r.Energy.TDEE)
	}
	if len(r.Energy.Levels) != 5 {
		t.Fatalf("levels = %d", len(r.Energy.Levels))
	}
	if r.WaistToHip == nil || *r.WaistToHip != 0.9 {
		t.Fatalf("WHR = %v", r.WaistToHip)
	}
	if r.WaistToHeight == nil || *r.WaistToHeight != 0.5 {
		t.Fatalf("WHtR = %v", r.WaistToHeight)
	}
	if r.Disclaimer != Disclaimer || !strings.Contains(strings.ToLower(r.Disclaimer), "estimativa") {
		t.Fatal("every result carries the estimate disclaimer")
	}
}

func TestEstimateFemaleNeedsHipForNavy(t *testing.T) {
	r, err := Estimate(femaleInput())
	if err != nil {
		t.Fatal(err)
	}
	if navy, ok := method(r, MethodNavy); !ok || navy != 28.9 {
		t.Fatalf("navy = %v %v, want 28.9", navy, ok)
	}
	in := femaleInput()
	delete(in.Circumferences, Hip)
	r, err = Estimate(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := method(r, MethodNavy); ok {
		t.Fatal("navy for women needs the hip")
	}
	if r.WaistToHip != nil {
		t.Fatal("no hip, no waist-to-hip ratio")
	}
}

func TestEstimateWithSkinfolds(t *testing.T) {
	in := maleInput()
	in.Skinfolds = map[Field]float64{FoldChest: 15, FoldAbdomen: 25, FoldThigh: 20} // sum 60
	r, err := Estimate(in)
	if err != nil {
		t.Fatal(err)
	}
	if jp, ok := method(r, MethodJacksonP3); !ok || jp != 17.9 {
		t.Fatalf("jp3 = %v %v, want 17.9", jp, ok)
	}
	if len(r.Methods) != 4 {
		t.Fatalf("methods = %d, want 4", len(r.Methods))
	}
	// A skinfold set missing one site is ignored, not guessed.
	in.Skinfolds = map[Field]float64{FoldChest: 15, FoldAbdomen: 25}
	r, _ = Estimate(in)
	if _, ok := method(r, MethodJacksonP3); ok {
		t.Fatal("two of three skinfolds must not run the equation")
	}
	// Women use triceps, suprailiac and thigh.
	f := femaleInput()
	f.Skinfolds = map[Field]float64{FoldTriceps: 20, FoldSuprailiac: 20, FoldThigh: 20}
	r, _ = Estimate(f)
	if jp, ok := method(r, MethodJacksonP3); !ok || jp != 24.1 {
		t.Fatalf("female jp3 = %v %v, want 24.1", jp, ok)
	}
}

func TestCombineAgreement(t *testing.T) {
	cases := []struct {
		vals   []float64
		median float64
		spread float64
		level  Agreement
	}{
		{[]float64{20}, 20, 0, AgreementSingle},
		{[]float64{20, 22}, 21, 2, AgreementHigh},
		{[]float64{18, 22, 25}, 22, 7, AgreementLow},
		{[]float64{18, 21, 23}, 21, 5, AgreementMedium},
		{[]float64{10, 20, 30, 40}, 25, 30, AgreementLow},
	}
	for _, c := range cases {
		var ms []BodyFat
		for _, v := range c.vals {
			ms = append(ms, BodyFat{Percent: v})
		}
		m, s, l := combine(ms)
		if m != c.median || s != c.spread || l != c.level {
			t.Errorf("combine(%v) = %v %v %v, want %v %v %v", c.vals, m, s, l, c.median, c.spread, c.level)
		}
	}
}

func TestImplausibleMethodIsDroppedNotAveraged(t *testing.T) {
	// Waist 60 with neck 40 and height 190: Navy and RFM both come out below 2%, so
	// only CUN-BAE may speak, and the result says why the others were left out.
	in := Input{Sex: Male, AgeYears: 30, HeightCM: 190, WeightKG: 70, Circumferences: map[Field]float64{Neck: 40, Waist: 60}}
	r, err := Estimate(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Methods) != 1 || r.Methods[0].Method != MethodCUNBAE || r.Agreement != AgreementSingle {
		t.Fatalf("methods = %+v agreement = %s", r.Methods, r.Agreement)
	}
	if len(r.Warnings) != 2 {
		t.Fatalf("warnings = %q", r.Warnings)
	}
}

func TestNoUsableEstimate(t *testing.T) {
	in := Input{Sex: Male, AgeYears: 18, HeightCM: 180, WeightKG: 40.5, Circumferences: map[Field]float64{Neck: 35, Waist: 55}}
	_, err := Estimate(in)
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Issues[0].Code != "no_estimate" {
		t.Fatalf("want no_estimate, got %v", err)
	}
}

func TestEstimateRefusesInvalidInput(t *testing.T) {
	cases := map[string]func(*Input){
		"minor":         func(i *Input) { i.AgeYears = 17 },
		"no sex":        func(i *Input) { i.Sex = "" },
		"no waist":      func(i *Input) { delete(i.Circumferences, Waist) },
		"waist<=neck":   func(i *Input) { i.Circumferences[Waist] = 38 },
		"height range":  func(i *Input) { i.HeightCM = 17.5 },
		"weight range":  func(i *Input) { i.WeightKG = 800 },
		"bmi":           func(i *Input) { i.WeightKG = 31 },
		"waist>=height": func(i *Input) { i.Circumferences[Waist] = 185 },
	}
	for name, mutate := range cases {
		in := maleInput()
		mutate(&in)
		_, err := Estimate(in)
		var verr *ValidationError
		if !errors.As(err, &verr) || len(verr.Issues) == 0 {
			t.Errorf("%s: want a ValidationError, got %v", name, err)
			continue
		}
		for _, i := range verr.Issues {
			if i.Severity != SeverityError || i.Message == "" {
				t.Errorf("%s: bad issue %+v", name, i)
			}
		}
	}
}

func TestResultJSONRoundTrip(t *testing.T) {
	r, err := Estimate(maleInput())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back Result
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.BodyFatPercent != r.BodyFatPercent || len(back.Methods) != len(r.Methods) || back.Input.Circumferences[Waist] != 90 {
		t.Fatalf("round trip lost data: %+v", back)
	}
}

func TestCompare(t *testing.T) {
	a, _ := Estimate(maleInput())
	in := maleInput()
	in.WeightKG = 77
	in.Circumferences[Waist] = 85
	b, err := Estimate(in)
	if err != nil {
		t.Fatal(err)
	}
	d := Compare(a, b, 30)
	if d.WeightKG != -3 || d.WaistCM != -5 || d.Days != 30 {
		t.Fatalf("delta = %+v", d)
	}
	if d.BodyFatPP >= 0 {
		t.Fatalf("a smaller waist must lower the estimate, got %v", d.BodyFatPP)
	}
	lines := d.Summary()
	if len(lines) != 4 || !strings.Contains(lines[2], "Peso: -3 kg") || !strings.Contains(lines[3], "Cintura: -5 cm") {
		t.Fatalf("summary = %q", lines)
	}
	same := Compare(a, a, 1)
	if !same.FatWithinNoise || !strings.Contains(same.Summary()[0], "sem mudança clara") {
		t.Fatalf("identical results must read as no clear change: %q", same.Summary())
	}
}

func TestFormatResultIsLabelledAsEstimate(t *testing.T) {
	r, _ := Estimate(maleInput())
	text := FormatResult(r)
	for _, want := range []string{"estimativa", "Gordura corporal", "Metabolismo basal", "Não é diagnóstico"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Errorf("result text lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "diagnóstico:") || strings.Contains(text, "%%") {
		t.Fatalf("bad formatting:\n%s", text)
	}
}

func TestMethodsAreDocumented(t *testing.T) {
	for _, m := range []Method{MethodNavy, MethodRFM, MethodCUNBAE, MethodJacksonP3} {
		i, ok := Info(m)
		if !ok || i.Name == "" || i.Plain == "" || i.Limits == "" || i.Citation == "" || i.Needs == "" {
			t.Errorf("method %s is not fully documented: %+v", m, i)
		}
	}
	if len(Methods()) != 4 {
		t.Fatal("expected four documented methods")
	}
}
