package unit_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	bodycomp "github.com/hyz-is/arandu-bodycomp"
)

// The tests in this file call the package the way an application does, from
// outside it, so a symbol the README teaches is a symbol that is exported.

func sampleInput() bodycomp.Input {
	return bodycomp.Input{
		Sex: bodycomp.Male, AgeYears: 30, HeightCM: 180, WeightKG: 80,
		Activity: bodycomp.Moderate,
		Circumferences: map[bodycomp.Field]float64{
			bodycomp.Neck: 40, bodycomp.Waist: 90, bodycomp.Hip: 100,
		},
	}
}

func TestEstimateAnswersWithAnEstimateAndTheDisclaimer(t *testing.T) {
	t.Parallel()

	result, err := bodycomp.Estimate(sampleInput())
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if result.BodyFatPercent <= 2 || result.BodyFatPercent >= 65 {
		t.Errorf("body fat = %.1f, want a plausible percentage", result.BodyFatPercent)
	}
	if len(result.Methods) == 0 {
		t.Error("no method ran")
	}
	if !strings.Contains(result.Disclaimer, "diagnóstico") {
		t.Errorf("disclaimer = %q, want the not-a-diagnosis notice", result.Disclaimer)
	}
	if bodycomp.FormatResult(result) == "" {
		t.Error("FormatResult returned nothing")
	}
}

func TestEstimateRefusesWhatItCannotEstimate(t *testing.T) {
	t.Parallel()

	bad := sampleInput()
	bad.HeightCM = 18

	_, err := bodycomp.Estimate(bad)
	var invalid *bodycomp.ValidationError
	if !errors.As(err, &invalid) || len(invalid.Issues) == 0 {
		t.Fatalf("Estimate(height 18 cm) = %v, want a *ValidationError with issues", err)
	}
}

func TestTheSameInputGivesTheSameEstimate(t *testing.T) {
	t.Parallel()

	first, err1 := bodycomp.Estimate(sampleInput())
	second, err2 := bodycomp.Estimate(sampleInput())
	if err1 != nil || err2 != nil {
		t.Fatalf("Estimate: %v, %v", err1, err2)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("two estimates of the same input differ")
	}
}

func TestCompareReadsTheChangeBetweenTwoEstimates(t *testing.T) {
	t.Parallel()

	before, err := bodycomp.Estimate(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	later := sampleInput()
	later.WeightKG = 77
	later.Circumferences = map[bodycomp.Field]float64{bodycomp.Neck: 40, bodycomp.Waist: 86, bodycomp.Hip: 100}
	after, err := bodycomp.Estimate(later)
	if err != nil {
		t.Fatal(err)
	}

	delta := bodycomp.Compare(before, after, 30)
	if delta.WeightKG != -3 {
		t.Errorf("weight change = %v, want -3", delta.WeightKG)
	}
	if len(delta.Summary()) == 0 {
		t.Error("Summary is empty")
	}
}

func TestTheParsersReadWhatAPersonTypes(t *testing.T) {
	t.Parallel()

	if cm, err := bodycomp.ParseLengthCM("1,75 m"); err != nil || cm != 175 {
		t.Errorf("ParseLengthCM(1,75 m) = %v, %v; want 175", cm, err)
	}
	if kg, err := bodycomp.ParseMassKG("70 kg"); err != nil || kg != 70 {
		t.Errorf("ParseMassKG(70 kg) = %v, %v; want 70", kg, err)
	}
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	if years, err := bodycomp.ParseAge("15/03/1990", now); err != nil || years != 36 {
		t.Errorf("ParseAge(15/03/1990) = %v, %v; want 36", years, err)
	}
	if sex, err := bodycomp.ParseSex("mulher"); err != nil || sex != bodycomp.Female {
		t.Errorf("ParseSex(mulher) = %v, %v; want female", sex, err)
	}
}

func TestAMinorEndsTheConversationAndKeepsNothing(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	session := bodycomp.NewSession(now, bodycomp.Prefill{})
	if session.Start().Text == "" {
		t.Fatal("Start asked nothing")
	}

	var reply bodycomp.Reply
	for _, answer := range []string{"homem", "12/05/2015"} {
		reply = session.Answer(answer, now)
		if reply.Cancelled {
			break
		}
	}
	if !reply.Cancelled || reply.Reason != "minor" || reply.Result != nil {
		t.Errorf("reply for a minor = %+v, want cancelled with reason minor and no result", reply)
	}
}

func TestEveryMeasurementHasATextForTheScreen(t *testing.T) {
	t.Parallel()

	if len(bodycomp.Methods()) == 0 || len(bodycomp.Specs()) == 0 {
		t.Fatal("Methods or Specs is empty")
	}
	for _, method := range bodycomp.Methods() {
		if strings.TrimSpace(method.Citation) == "" {
			t.Errorf("method %s has no citation", method.Method)
		}
	}
}
