package bodycomp

import "testing"

func TestValidateValueSuggestions(t *testing.T) {
	cases := []struct {
		f       Field
		v       float64
		suggest float64 // 0 means a plain refusal
		ok      bool
	}{
		{Waist, 85, 0, true},
		{Waist, 850, 85, false}, // a swallowed decimal point
		{Waist, 8.5, 85, false}, // a lost zero
		{FieldHeight, 17.5, 175, false},
		{FieldWeight, 7.2, 72, false},
		{FieldWeight, 720, 72, false},
		{Neck, 150, 0, false},    // no slip makes it plausible: plain refusal
		{Waist, 32, 81.3, false}, // inches typed as centimetres
		{Waist, -5, 0, false},
	}
	for _, c := range cases {
		field := c.f
		issues := ValidateValue(field, c.v)
		if c.ok {
			if len(issues) != 0 {
				t.Errorf("%s %v: unexpected %v", field, c.v, issues)
			}
			continue
		}
		if len(issues) != 1 || issues[0].Severity != SeverityError || issues[0].Message == "" {
			t.Errorf("%s %v: want one error, got %v", field, c.v, issues)
			continue
		}
		got := issues[0].Suggestion
		switch {
		case c.suggest == 0 && got != nil:
			t.Errorf("%s %v: unexpected suggestion %v", field, c.v, *got)
		case c.suggest != 0 && (got == nil || *got != c.suggest):
			t.Errorf("%s %v: suggestion = %v, want %v", field, c.v, got, c.suggest)
		}
	}
}

func TestValidateAge(t *testing.T) {
	if got := ValidateAge(17); len(got) != 1 || got[0].Code != "minor" {
		t.Fatalf("17 must be refused as a minor: %v", got)
	}
	if got := ValidateAge(18); len(got) != 0 {
		t.Fatalf("18 is allowed: %v", got)
	}
	if got := ValidateAge(120); len(got) != 1 || got[0].Code != "age_range" {
		t.Fatalf("120: %v", got)
	}
	if got := ValidateAge(0); len(got) != 1 {
		t.Fatalf("0: %v", got)
	}
}

func TestCheckRelation(t *testing.T) {
	have := map[Field]float64{Neck: 40, ArmRelaxed: 32, Thigh: 55}
	if got := CheckRelation(Waist, 38, 180, have); len(got) < 1 || got[0].Code != "waist_le_neck" {
		t.Fatalf("waist below neck: %v", got)
	}
	if got := CheckRelation(Waist, 85, 180, have); len(got) != 0 {
		t.Fatalf("a normal waist: %v", got)
	}
	if got := CheckRelation(ArmFlexed, 30, 180, have); len(got) != 1 || got[0].Code != "flexed_lt_relaxed" || got[0].Severity != SeverityError {
		t.Fatalf("flexed below relaxed: %v", got)
	}
	if got := CheckRelation(Calf, 60, 180, have); len(got) != 1 || got[0].Severity != SeverityWarning {
		t.Fatalf("calf above thigh is a warning: %v", got)
	}
	if got := CheckRelation(Waist, 185, 180, have); len(got) != 1 || got[0].Code != "waist_ge_height" {
		t.Fatalf("waist above height: %v", got)
	}
}

func TestValidateIsStableAndErrorsFirst(t *testing.T) {
	in := maleInput()
	in.AgeYears = 17
	in.Circumferences[Calf] = 60
	in.Circumferences[Thigh] = 50
	got := Validate(in)
	if len(got) < 2 || got[0].Severity != SeverityError || got[len(got)-1].Severity != SeverityWarning {
		t.Fatalf("errors must come first: %v", got)
	}
	if !HasErrors(got) {
		t.Fatal("HasErrors must see the minor")
	}
}
