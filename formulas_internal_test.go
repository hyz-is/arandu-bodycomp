package bodycomp

import (
	"math"
	"testing"
)

func near(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s = %.4f, want %.4f (±%g)", name, got, want, tol)
	}
}

// The expected values below were computed independently (awk, double
// precision) from the equations as printed in the cited publications.

func TestNavyMale(t *testing.T) {
	// Hodgdon & Beckett 1984: height 180, waist 90, neck 40.
	near(t, "navy male", NavyMale(90, 40, 180), 18.3675, 0.001)
}

func TestNavyFemale(t *testing.T) {
	// height 165, waist 75, hip 100, neck 34.
	near(t, "navy female", NavyFemale(75, 100, 34, 165), 28.9359, 0.001)
}

// The metric forms must be the same equations as the inch forms printed in the
// Navy regulation (OPNAVINST 6110.1), to within the rounding of their constants.
func TestNavyMetricMatchesInchForm(t *testing.T) {
	const in = 2.54
	for _, c := range []struct{ waist, neck, height float64 }{{90, 40, 180}, {100, 42, 175}, {80, 36, 170}, {110, 45, 190}} {
		metric := NavyMale(c.waist, c.neck, c.height)
		inch := navyMaleInches(c.waist/in, c.neck/in, c.height/in)
		near(t, "navy male metric vs inch", metric, inch, 0.6)
	}
	for _, c := range []struct{ waist, hip, neck, height float64 }{{75, 100, 34, 165}, {85, 105, 35, 160}, {70, 95, 32, 170}} {
		metric := NavyFemale(c.waist, c.hip, c.neck, c.height)
		inch := navyFemaleInches(c.waist/in, c.hip/in, c.neck/in, c.height/in)
		near(t, "navy female metric vs inch", metric, inch, 0.6)
	}
}

func TestRFM(t *testing.T) {
	// 64 - 20*(175/85) = 22.82 ; 76 - 20*(165/75) = 32.00
	near(t, "rfm male", RFM(Male, 175, 85), 22.8235, 0.001)
	near(t, "rfm female", RFM(Female, 165, 75), 32.0, 0.001)
}

func TestCUNBAE(t *testing.T) {
	near(t, "cun-bae male 40y BMI 25", CUNBAE(Male, 40, 25), 23.4320, 0.001)
	near(t, "cun-bae female 30y BMI 22", CUNBAE(Female, 30, 22), 29.4022, 0.001)
}

func TestJacksonPollock3(t *testing.T) {
	// Sum of three folds 60 mm, age 30.
	near(t, "jp3 male", JacksonPollock3(Male, 30, 60), 17.9453, 0.001)
	near(t, "jp3 female", JacksonPollock3(Female, 30, 60), 24.1279, 0.001)
}

func TestBMIFFMIEnergy(t *testing.T) {
	near(t, "bmi", BMI(80, 180), 24.6914, 0.001)
	near(t, "ffmi", FFMI(65, 180), 20.0617, 0.001)
	// At the 1.80 m reference height the normalisation is the identity.
	near(t, "normalised ffmi at 180", NormalizedFFMI(20, 180), 20, 1e-9)
	near(t, "normalised ffmi at 170", NormalizedFFMI(20, 170), 20.61, 1e-9)
	// Mifflin-St Jeor textbook case: 80 kg, 180 cm, 30 y, man.
	near(t, "mifflin male", MifflinStJeor(Male, 80, 180, 30), 1780, 1e-9)
	near(t, "mifflin female", MifflinStJeor(Female, 60, 165, 30), 1320.25, 1e-9)
	near(t, "katch", KatchMcArdle(65), 1774, 1e-9)
	near(t, "whtr", WaistToHeight(85, 170), 0.5, 1e-9)
	near(t, "whr", WaistToHip(90, 100), 0.9, 1e-9)
}

func TestActivityFactors(t *testing.T) {
	want := map[ActivityLevel]float64{Sedentary: 1.2, Light: 1.375, Moderate: 1.55, Intense: 1.725, VeryIntense: 1.9}
	for l, f := range want {
		if ActivityFactor(l) != f {
			t.Fatalf("factor %s = %v, want %v", l, ActivityFactor(l), f)
		}
		if ActivityLabel(l) == "" {
			t.Fatalf("no label for %s", l)
		}
	}
	if ActivityFactor("nope") != 0 {
		t.Fatal("unknown level must have factor 0")
	}
	for in, want := range map[string]ActivityLevel{"1": Sedentary, "Leve": Light, "3": Moderate, "intensa": Intense, "5": VeryIntense} {
		if got, ok := ParseActivity(in); !ok || got != want {
			t.Fatalf("ParseActivity(%q) = %v %v", in, got, ok)
		}
	}
	if _, ok := ParseActivity("9"); ok {
		t.Fatal("9 is not a level")
	}
}

func TestBandsAndClasses(t *testing.T) {
	cases := []struct {
		sex  Sex
		pct  float64
		want string
	}{
		{Male, 4, "gordura essencial"}, {Male, 10, "faixa de atleta"}, {Male, 16, "faixa fitness"},
		{Male, 20, "faixa média"}, {Male, 30, "acima da faixa média"},
		{Female, 12, "gordura essencial"}, {Female, 18, "faixa de atleta"}, {Female, 23, "faixa fitness"},
		{Female, 28, "faixa média"}, {Female, 35, "acima da faixa média"},
	}
	for _, c := range cases {
		if got := FatCategory(c.sex, c.pct); got != c.want {
			t.Errorf("FatCategory(%s, %v) = %q, want %q", c.sex, c.pct, got, c.want)
		}
	}
	if got := BMIClass(24.9); got != "dentro da faixa de referência (18,5 a 24,9)" {
		t.Fatalf("BMIClass(24.9) = %q", got)
	}
}
