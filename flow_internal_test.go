package bodycomp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

// run feeds answers in order and returns the last reply.
func run(t *testing.T, s *Session, answers ...string) Reply {
	t.Helper()
	var r Reply
	for i, a := range answers {
		r = s.Answer(a, t0.Add(time.Duration(i)*time.Minute))
	}
	return r
}

func TestFlowMaleHappyPathCoreOnly(t *testing.T) {
	s := NewSession(t0, Prefill{})
	first := s.Start()
	if first.Step != "sex" || first.Progress.Index != 1 || first.Progress.Total != 7 {
		t.Fatalf("first = %+v", first)
	}
	if !strings.Contains(first.Text, "3 minutos") || !strings.Contains(first.Text, "fita métrica") {
		t.Fatalf("the opening states the time and the tool:\n%s", first.Text)
	}
	r := run(t, s, "homem", "30", "1,80", "80 kg", "40", "90", "3", "não")
	if !r.Done || r.Result == nil || s.Status != StatusDone {
		t.Fatalf("not done: %+v", r)
	}
	if r.Result.Input.HeightCM != 180 || r.Result.Input.Activity != Moderate {
		t.Fatalf("input = %+v", r.Result.Input)
	}
	if navy, ok := method(*r.Result, MethodNavy); !ok || navy != 18.4 {
		t.Fatalf("navy = %v", navy)
	}
	if !strings.Contains(r.Text, "estimativa") {
		t.Fatalf("result text:\n%s", r.Text)
	}
	// Once done, further messages do not restart anything.
	if again := s.Answer("oi", t0); !again.Done || again.Result == nil {
		t.Fatalf("a finished session answers with its result: %+v", again)
	}
}

func TestFlowFemaleAsksHipInCore(t *testing.T) {
	s := NewSession(t0, Prefill{})
	s.Start()
	r := run(t, s, "mulher", "30", "165", "60", "34", "75")
	if r.Step != "hip" || r.Progress.Total != 8 {
		t.Fatalf("a woman is asked the hip before the optional part: %+v", r)
	}
	if strings.Contains(r.Text, "pular") {
		t.Fatalf("the hip is required for women:\n%s", r.Text)
	}
	r = run(t, s, "100", "2", "n")
	if !r.Done {
		t.Fatalf("not done: %+v", r)
	}
	if navy, ok := method(*r.Result, MethodNavy); !ok || navy != 28.9 {
		t.Fatalf("navy female = %v %v", navy, ok)
	}
}

func TestFlowTipsAndUnits(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Male})
	r := s.Start()
	if r.Step != "age" {
		t.Fatalf("a prefilled sex is not asked again: %+v", r)
	}
	r = run(t, s, "15/03/1990")
	if r.Step != "height" || !strings.Contains(r.Text, "Anotado: 36 anos") {
		t.Fatalf("birth date: %+v", r)
	}
	r = run(t, s, "5'11")
	if !strings.Contains(r.Text, "Anotado: altura 180,3 cm") {
		t.Fatalf("feet and inches:\n%s", r.Text)
	}
	r = run(t, s, "176 lb")
	if !strings.Contains(r.Text, "peso 79,8 kg") {
		t.Fatalf("pounds:\n%s", r.Text)
	}
	if r.Step != "neck" || !strings.Contains(r.Text, "Como medir") || !strings.Contains(r.Text, "gogó") {
		t.Fatalf("each measure comes with its tip:\n%s", r.Text)
	}
	// "ajuda" repeats the tip without moving on.
	again := run(t, s, "ajuda")
	if again.Step != "neck" || !strings.Contains(again.Text, "gogó") {
		t.Fatalf("help: %+v", again)
	}
	// The waist tip for women differs from men's.
	if m, _ := SpecFor(Waist); m.Tip(Male) == m.Tip(Female) {
		t.Fatal("waist tip must depend on sex")
	}
}

func TestFlowTypoSuggestionAndRefusal(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s.Start()
	run(t, s, "30", "80")
	// 700 cm neck: no slip makes it plausible, so it is asked again.
	r := run(t, s, "700")
	if r.Step != "neck" || !strings.Contains(r.Text, "fora do que é possível") {
		t.Fatalf("refusal: %+v", r)
	}
	// 4O.5 (letter O) is repaired and accepted.
	r = run(t, s, "4O,5")
	if r.Step != "waist" || !strings.Contains(r.Text, "pescoço 40,5 cm") {
		t.Fatalf("typo repair: %+v", r)
	}
	// 900 mm... wait for the decimal slip: 850 means 85.
	r = run(t, s, "850")
	if r.Step != "waist" || s.Pending == nil || !strings.Contains(r.Text, "Você quis dizer 85 cm") {
		t.Fatalf("suggestion: %+v pending=%v", r, s.Pending)
	}
	r = run(t, s, "sim")
	if r.Step != "activity" || s.Circumferences[Waist] != 85 {
		t.Fatalf("accepting the suggestion: %+v waist=%v", r, s.Circumferences[Waist])
	}
	// A new number instead of "sim" is read as the new answer.
	s2 := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s2.Start()
	run(t, s2, "30", "80", "40", "850")
	r = run(t, s2, "88")
	if s2.Circumferences[Waist] != 88 || r.Step != "activity" {
		t.Fatalf("a fresh number replaces the suggestion: %+v", r)
	}
	// "não" asks again.
	s3 := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s3.Start()
	run(t, s3, "30", "80", "40", "850")
	r = run(t, s3, "não")
	if r.Step != "waist" || s3.Pending != nil {
		t.Fatalf("declining: %+v", r)
	}
}

func TestFlowRelationsAreCheckedAtEntry(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s.Start()
	run(t, s, "30", "80", "55")
	r := run(t, s, "54")
	if r.Step != "waist" || !strings.Contains(r.Text, "menor que o pescoço") {
		t.Fatalf("waist below neck: %+v", r)
	}
	if _, kept := s.Circumferences[Waist]; kept {
		t.Fatal("a refused value must not be stored")
	}
	// Weight and height that give an absurd BMI are refused at the weight.
	s = NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s.Start()
	r = run(t, s, "30", "31")
	if r.Step != "weight" || !strings.Contains(r.Text, "IMC") {
		t.Fatalf("BMI check: %+v", r)
	}
}

func TestFlowWarningNeedsConfirmation(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Female, HeightCM: 165})
	s.Start()
	run(t, s, "30", "60", "34", "75")
	// Hip far larger than waist: odd, but a woman may confirm.
	r := run(t, s, "140")
	if s.Pending == nil || r.Step != "hip" || !strings.Contains(r.Text, `"sim"`) {
		t.Fatalf("odd ratio asks to confirm: %+v", r)
	}
	r = run(t, s, "sim")
	if s.Circumferences[Hip] != 140 || r.Step != "activity" {
		t.Fatalf("confirmed: %+v", r)
	}
}

func TestFlowSkipBackAndCancel(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s.Start()
	r := run(t, s, "30", "80", "40", "90")
	if r.Step != "activity" {
		t.Fatalf("step = %s", r.Step)
	}
	// Required measures cannot be skipped.
	s2 := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s2.Start()
	run(t, s2, "30", "80")
	r2 := run(t, s2, "pular")
	if r2.Step != "neck" || !strings.Contains(r2.Text, "necessária") {
		t.Fatalf("required skip: %+v", r2)
	}
	// Optional measures can.
	r = run(t, s, "pular")
	if r.Step != "more" || !s.Skipped["activity"] {
		t.Fatalf("skip activity: %+v", r)
	}
	if !strings.Contains(r.Text, "4 minutos") {
		t.Fatalf("the gate states how long the rest takes:\n%s", r.Text)
	}
	// Back undoes the last answer (the skip), then the waist.
	r = run(t, s, "voltar")
	if r.Step != "activity" || s.Skipped["activity"] {
		t.Fatalf("back: %+v", r)
	}
	r = run(t, s, "voltar")
	if r.Step != "waist" {
		t.Fatalf("back again: %+v", r)
	}
	if _, kept := s.Circumferences[Waist]; kept {
		t.Fatal("back must forget the value")
	}
	// Back at the very start is harmless.
	s3 := NewSession(t0, Prefill{})
	s3.Start()
	if r := run(t, s3, "voltar"); r.Step != "sex" {
		t.Fatalf("back at start: %+v", r)
	}
	// Cancel ends the session without a result.
	r = run(t, s, "cancelar")
	if !r.Cancelled || s.Status != StatusCancelled || r.Result != nil {
		t.Fatalf("cancel: %+v", r)
	}
}

func TestFlowRefusesMinorsAndKeepsNothing(t *testing.T) {
	s := NewSession(t0, Prefill{})
	s.Start()
	r := run(t, s, "homem", "16")
	if !r.Cancelled || r.Reason != ReasonMinor || s.AgeYears != 0 || s.Status != StatusCancelled {
		t.Fatalf("minor: %+v session=%+v", r, s)
	}
	s = NewSession(t0, Prefill{})
	s.Start()
	r = run(t, s, "mulher", "10/05/2015")
	if !r.Cancelled || r.Reason != ReasonMinor {
		t.Fatalf("minor by birth date: %+v", r)
	}
}

func TestFlowResumeAfterJSONRoundTrip(t *testing.T) {
	s := NewSession(t0, Prefill{})
	s.Start()
	run(t, s, "homem", "30", "180", "80", "40")
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back Session
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	r := back.Resume()
	if r.Step != "waist" || !strings.Contains(r.Text, "continuar de onde paramos") || r.Progress.Index != 6 {
		t.Fatalf("resume: %+v", r)
	}
	r = back.Answer("90", t0.Add(72*time.Hour))
	if r.Step != "activity" || back.Circumferences[Waist] != 90 {
		t.Fatalf("answering after the round trip: %+v", r)
	}
	// An empty (zero) session loaded from storage is usable.
	var zero Session
	if r := zero.Start(); r.Step != "sex" {
		t.Fatalf("zero session: %+v", r)
	}
}

func TestFlowOptionalCircumferencesAndCaliper(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s.Start()
	r := run(t, s, "30", "80", "40", "90", "3", "sim")
	if r.Step != "hip" || r.Progress.Total <= 8 {
		t.Fatalf("the optional part opens with the hip for men: %+v", r)
	}
	for _, a := range []string{"100", "92", "100", "115", "34", "37", "29", "58", "38"} {
		r = run(t, s, a)
	}
	if r.Step != "caliper" {
		t.Fatalf("after the circumferences comes the caliper gate: %+v", r)
	}
	r = run(t, s, "talvez")
	if r.Step != "caliper" {
		t.Fatalf("a gate needs yes or no: %+v", r)
	}
	r = run(t, s, "tenho")
	if r.Step != "fold_chest" {
		t.Fatalf("caliper: %+v", r)
	}
	r = run(t, s, "15", "25 mm", "2 cm")
	if !r.Done {
		t.Fatalf("not done: %+v", r)
	}
	if _, ok := method(*r.Result, MethodJacksonP3); !ok {
		t.Fatal("skinfolds must add the Jackson-Pollock estimate")
	}
	if r.Result.Input.Skinfolds[FoldThigh] != 20 || r.Result.Input.Circumferences[Thigh] != 58 {
		t.Fatalf("input = %+v", r.Result.Input)
	}
	// Reordering: the arm measures were relaxed (34) then flexed (37).
	if r.Result.Input.Circumferences[ArmFlexed] != 37 {
		t.Fatalf("circumferences = %v", r.Result.Input.Circumferences)
	}
}

func TestFlowFlexedArmSmallerThanRelaxedIsRefused(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s.Start()
	run(t, s, "30", "80", "40", "90", "3", "sim", "100", "92", "100", "115", "34")
	r := run(t, s, "30") // flexed
	if r.Step != "arm_flexed" || !strings.Contains(r.Text, "menor que o relaxado") {
		t.Fatalf("flexed: %+v", r)
	}
}

func TestFlowFinishSendsBackToTheValueToBlame(t *testing.T) {
	s := NewSession(t0, Prefill{Sex: Male, HeightCM: 180})
	s.Start()
	run(t, s, "30", "80")
	// A value that slipped past entry checks: waist below neck.
	s.Circumferences[Neck] = 55
	s.Circumferences[Waist] = 54
	s.History = append(s.History, "neck", "waist")
	s.Current = "activity"
	s.Gates[stepMore] = false
	r := s.Answer("pular", t0)
	if r.Done || r.Step != "waist" || !strings.Contains(r.Text, "menor que o pescoço") {
		t.Fatalf("finish must return to the waist: %+v", r)
	}
	if _, kept := s.Circumferences[Waist]; kept {
		t.Fatal("the blamed value is forgotten")
	}
}

func TestEveryAskedFieldHasATip(t *testing.T) {
	for _, sp := range Specs() {
		if sp.Tip(Male) == "" || sp.Label == "" || sp.Min <= 0 || sp.Max <= sp.Min {
			t.Errorf("incomplete spec for %s: %+v", sp.Field, sp)
		}
	}
}
