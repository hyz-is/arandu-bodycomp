package bodycomp

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Status is where a Session stands.
type Status string

// Session statuses.
const (
	StatusActive    Status = "active"
	StatusDone      Status = "done"
	StatusCancelled Status = "cancelled"
)

// Step identifiers. A step that collects a Field uses the Field's own name.
const (
	stepActivity = "activity"
	stepMore     = "more"    // gate: collect the optional circumferences?
	stepCaliper  = "caliper" // gate: collect skinfolds?
)

// Reasons a Reply may carry so the channel can react without parsing text.
const (
	ReasonMinor = "minor" // the person is under 18: nothing is kept
)

// Pending is a value waiting for a "sim": either a correction the engine
// proposes (a lost decimal point) or an odd-but-possible value to confirm.
type Pending struct {
	Step  string  `json:"step"`
	Value float64 `json:"value"`
}

// Session is the guided measurement conversation as plain data. It holds no
// clock, channel or storage: the caller loads it, calls Answer with the next
// message and saves it again, so a person can stop and resume days later.
// It marshals to JSON.
type Session struct {
	Status Status `json:"status"`

	Sex            Sex               `json:"sex,omitempty"`
	AgeYears       int               `json:"age_years,omitempty"`
	HeightCM       float64           `json:"height_cm,omitempty"`
	WeightKG       float64           `json:"weight_kg,omitempty"`
	Activity       ActivityLevel     `json:"activity,omitempty"`
	Circumferences map[Field]float64 `json:"circumferences,omitempty"`
	Skinfolds      map[Field]float64 `json:"skinfolds,omitempty"`
	Skipped        map[string]bool   `json:"skipped,omitempty"`
	Gates          map[string]bool   `json:"gates,omitempty"`
	Current        string            `json:"current,omitempty"`
	History        []string          `json:"history,omitempty"`
	Pending        *Pending          `json:"pending,omitempty"`
	StartedAt      time.Time         `json:"started_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Result         *Result           `json:"result,omitempty"`
}

// Prefill carries what the app already knows about the person, so a later
// measurement does not ask it again. Zero values are unknown.
type Prefill struct {
	Sex      Sex
	HeightCM float64
	AgeYears int
	Activity ActivityLevel
}

// Progress tells where the person is: question Index of Total.
type Progress struct{ Index, Total int }

// Reply is what the channel sends back and what it should do next.
type Reply struct {
	// Text is the message to send, in pt-BR.
	Text string
	// Step is the step now being asked; empty when the session ended.
	Step     string
	Progress Progress
	// Done is true when Result is set. Cancelled is true when the person
	// stopped or was refused; Reason says why when it is a refusal.
	Done      bool
	Cancelled bool
	Reason    string
	Result    *Result
}

// NewSession starts a conversation at now. It does not produce the first
// question: call Start for that.
func NewSession(now time.Time, pre Prefill) *Session {
	s := &Session{
		Status:         StatusActive,
		Circumferences: map[Field]float64{},
		Skinfolds:      map[Field]float64{},
		Skipped:        map[string]bool{},
		Gates:          map[string]bool{},
		StartedAt:      now,
		UpdatedAt:      now,
	}
	if pre.Sex.Valid() {
		s.Sex = pre.Sex
	}
	if pre.HeightCM > 0 && len(ValidateValue(FieldHeight, pre.HeightCM)) == 0 {
		s.HeightCM = pre.HeightCM
	}
	if pre.AgeYears >= MinAgeYears && pre.AgeYears <= MaxAgeYears {
		s.AgeYears = pre.AgeYears
	}
	if ActivityFactor(pre.Activity) > 0 {
		s.Activity = pre.Activity
	}
	return s
}

// Input assembles the estimate input from what has been collected.
func (s *Session) Input() Input {
	in := Input{Sex: s.Sex, AgeYears: s.AgeYears, HeightCM: s.HeightCM, WeightKG: s.WeightKG, Activity: s.Activity,
		Circumferences: map[Field]float64{}, Skinfolds: map[Field]float64{}}
	for f, v := range s.Circumferences {
		in.Circumferences[f] = v
	}
	for f, v := range s.Skinfolds {
		in.Skinfolds[f] = v
	}
	return in
}

// Start returns the opening message and first question.
func (s *Session) Start() Reply {
	s.ensure()
	s.Current = s.next()
	intro := "Vou fazer a sua bioimpedância digital: uma estimativa da gordura corporal a partir das suas medidas. " +
		"Pergunto uma medida por vez. O essencial leva uns 3 minutos e você precisa de uma fita métrica. " +
		"Pode parar quando quiser: eu guardo onde paramos."
	return s.ask(intro)
}

// Resume re-asks the current question after a pause.
func (s *Session) Resume() Reply {
	s.ensure()
	if s.Status != StatusActive {
		return Reply{Text: "Essa medição já terminou.", Done: s.Status == StatusDone, Cancelled: s.Status == StatusCancelled, Result: s.Result}
	}
	if s.Current == "" {
		s.Current = s.next()
	}
	return s.ask("Vamos continuar de onde paramos.")
}

// Answer feeds the person's next message to the conversation.
func (s *Session) Answer(text string, now time.Time) Reply {
	s.ensure()
	s.UpdatedAt = now
	if s.Status != StatusActive {
		return Reply{Text: "Essa medição já terminou. Para medir de novo, peça uma nova bioimpedância.",
			Done: s.Status == StatusDone, Cancelled: s.Status == StatusCancelled, Result: s.Result}
	}
	if s.Current == "" {
		s.Current = s.next()
	}
	word := strings.Trim(normalise(text), " .!?")

	switch word {
	case "cancelar", "sair", "parar", "desistir", "cancela":
		s.Status = StatusCancelled
		return Reply{Cancelled: true,
			Text: "Tudo bem, parei por aqui. O que você já mandou não vira resultado. Para medir de novo, é só pedir a bioimpedância."}
	case "ajuda", "dica", "como medir", "como", "":
		return s.ask("")
	case "voltar", "anterior":
		return s.back()
	case "pular", "pula":
		return s.skip()
	case "nao tenho", "n tenho":
		if !isGate(s.Current) {
			return s.skip()
		}
	}

	if s.Pending != nil {
		p := *s.Pending
		s.Pending = nil
		switch {
		case isYes(word):
			return s.commit(p.Step, p.Value)
		case isNo(word):
			return s.ask("Certo, vamos de novo.")
		}
		// Anything else is a fresh answer to the same question.
	}
	return s.handle(s.Current, text, now)
}

// ensure makes the maps usable after a JSON round trip of an empty session.
func (s *Session) ensure() {
	if s.Status == "" {
		s.Status = StatusActive
	}
	if s.Circumferences == nil {
		s.Circumferences = map[Field]float64{}
	}
	if s.Skinfolds == nil {
		s.Skinfolds = map[Field]float64{}
	}
	if s.Skipped == nil {
		s.Skipped = map[string]bool{}
	}
	if s.Gates == nil {
		s.Gates = map[string]bool{}
	}
}

// plan lists the steps in order for what is known so far.
func (s *Session) plan() []string {
	steps := []string{string(FieldSex), string(FieldAge), string(FieldHeight), string(FieldWeight), string(Neck), string(Waist)}
	if s.Sex == Female {
		steps = append(steps, string(Hip))
	}
	steps = append(steps, stepActivity, stepMore)
	if !s.Gates[stepMore] {
		return steps
	}
	if s.Sex != Female {
		steps = append(steps, string(Hip))
	}
	steps = append(steps, string(Abdomen), string(Chest), string(Shoulders), string(ArmRelaxed),
		string(ArmFlexed), string(Forearm), string(Thigh), string(Calf), stepCaliper)
	if s.Gates[stepCaliper] {
		for _, f := range FoldsFor(s.Sex) {
			steps = append(steps, string(f))
		}
	}
	return steps
}

func (s *Session) isGateSet(g string) bool { _, ok := s.Gates[g]; return ok }

func (s *Session) answered(step string) bool {
	switch step {
	case string(FieldSex):
		return s.Sex.Valid()
	case string(FieldAge):
		return s.AgeYears > 0
	case string(FieldHeight):
		return s.HeightCM > 0
	case string(FieldWeight):
		return s.WeightKG > 0
	case stepActivity:
		return s.Activity != "" || s.Skipped[step]
	case stepMore, stepCaliper:
		return s.isGateSet(step)
	}
	f := Field(step)
	if f.IsFold() {
		_, ok := s.Skinfolds[f]
		return ok || s.Skipped[step]
	}
	_, ok := s.Circumferences[f]
	return ok || s.Skipped[step]
}

// next returns the first unanswered step, or "" when the plan is complete.
func (s *Session) next() string {
	for _, step := range s.plan() {
		if !s.answered(step) {
			return step
		}
	}
	return ""
}

func isGate(step string) bool { return step == stepMore || step == stepCaliper }

// optional reports whether a step may be skipped.
func (s *Session) optional(step string) bool {
	switch step {
	case string(FieldSex), string(FieldAge), string(FieldHeight), string(FieldWeight), string(Neck), string(Waist):
		return false
	case string(Hip):
		return s.Sex != Female
	}
	return !isGate(step)
}

// progress counts question steps (gates excluded) answered and in the plan.
func (s *Session) progress() Progress {
	total, done := 0, 0
	for _, step := range s.plan() {
		if isGate(step) {
			continue
		}
		total++
		if s.answered(step) {
			done++
		}
	}
	idx := done + 1
	if idx > total {
		idx = total
	}
	return Progress{Index: idx, Total: total}
}

// ask renders the current question, with an optional lead-in line.
func (s *Session) ask(lead string) Reply {
	step := s.Current
	p := s.progress()
	var b strings.Builder
	if lead != "" {
		b.WriteString(lead)
		b.WriteString("\n\n")
	}
	switch step {
	case stepMore:
		b.WriteString("Pronto, já dá para calcular. Quer mandar mais medidas (tórax, braço, coxa e outras) para acompanhar melhor a sua evolução? " +
			"Leva mais uns 4 minutos.\nResponda *sim* para continuar ou *não* para ver o resultado agora.")
	case stepCaliper:
		b.WriteString("Você tem um adipômetro (plicômetro)? Com ele, 3 dobras de pele melhoram a estimativa. Leva uns 2 minutos.\nResponda *sim* ou *não*.")
	default:
		fmt.Fprintf(&b, "*%s* (passo %d de %d)\n%s", stepTitle(step), p.Index, p.Total, s.question(step))
	}
	return Reply{Text: b.String(), Step: step, Progress: p}
}

func stepTitle(step string) string {
	switch step {
	case string(FieldSex):
		return "Sexo"
	case string(FieldAge):
		return "Idade"
	case string(FieldHeight):
		return "Altura"
	case string(FieldWeight):
		return "Peso"
	case stepActivity:
		return "Nível de atividade"
	}
	if spec, ok := SpecFor(Field(step)); ok {
		return capitalise(spec.Label)
	}
	return step
}

func (s *Session) question(step string) string {
	skip := ""
	if s.optional(step) {
		skip = "\nSe não der para medir, mande *pular*."
	}
	switch step {
	case string(FieldSex):
		return "As fórmulas são diferentes para homens e mulheres. Você é *homem* ou *mulher*?"
	case string(FieldAge):
		return "Qual é a sua idade? Pode mandar os anos (34) ou a data de nascimento (15/03/1990). Eu não guardo a data."
	case stepActivity:
		var b strings.Builder
		b.WriteString("Como é a sua rotina de atividade? Responda com o número:")
		for i, l := range ActivityLevels() {
			fmt.Fprintf(&b, "\n%d. %s", i+1, ActivityLabel(l))
		}
		b.WriteString(skip)
		return b.String()
	}
	spec, _ := SpecFor(Field(step))
	var ask string
	switch spec.Unit {
	case UnitKG:
		ask = "Qual é o seu peso? Pode mandar em kg (72,5) ou libras (160 lb)."
	case UnitMM:
		ask = fmt.Sprintf("Quanto deu a %s? Mande em milímetros (12).", spec.Label)
	default:
		if step == string(FieldHeight) {
			ask = "Qual é a sua altura? Pode mandar em cm (175) ou metros (1,75)."
		} else {
			ask = fmt.Sprintf("Quanto deu o(a) %s? Mande em cm (85) ou polegadas (33 in).", spec.Label)
		}
	}
	return ask + "\n_Como medir:_ " + spec.Tip(s.Sex) + skip
}

// handle parses one answer for the current step.
func (s *Session) handle(step, text string, now time.Time) Reply {
	switch step {
	case string(FieldSex):
		v, err := ParseSex(text)
		if err != nil {
			return s.ask("Não entendi. Responda *homem* ou *mulher*.")
		}
		return s.commitSex(v)
	case string(FieldAge):
		years, err := ParseAge(text, now)
		if err != nil {
			if errors.Is(err, ErrFutureBirth) {
				return s.ask("Essa data de nascimento está no futuro. Confira e mande de novo.")
			}
			return s.ask("Não entendi a idade. Mande só os anos (34) ou a data de nascimento (15/03/1990).")
		}
		if issues := ValidateAge(years); len(issues) > 0 {
			if issues[0].Code == "minor" {
				s.Status = StatusCancelled
				s.AgeYears = 0
				return Reply{Cancelled: true, Reason: ReasonMinor, Text: issues[0].Message}
			}
			return s.ask(issues[0].Message)
		}
		return s.commit(step, float64(years))
	case stepActivity:
		level, ok := ParseActivity(text)
		if !ok {
			return s.ask("Responda com um número de 1 a 5.")
		}
		return s.commitActivity(level)
	case stepMore, stepCaliper:
		switch word := strings.Trim(normalise(text), " .!?"); {
		case isYes(word):
			return s.commitGate(step, true)
		case isNo(word):
			return s.commitGate(step, false)
		}
		return s.ask("Responda *sim* ou *não*.")
	}

	var v float64
	var err error
	f := Field(step)
	switch {
	case step == string(FieldWeight):
		v, err = ParseMassKG(text)
	case f.IsFold():
		v, err = ParseSkinfoldMM(text)
	default:
		v, err = ParseLengthCM(text)
	}
	if err != nil {
		if errors.Is(err, ErrWrongUnit) {
			return s.ask("Essa unidade não serve aqui. Mande o valor em " + wantedUnits(step) + ".")
		}
		return s.ask("Não entendi esse número. Mande só o valor, por exemplo " + example(step) + ".")
	}
	return s.check(step, v)
}

func wantedUnits(step string) string {
	switch {
	case step == string(FieldWeight):
		return "kg ou libras"
	case Field(step).IsFold():
		return "milímetros"
	}
	return "cm, metros ou polegadas"
}

func example(step string) string {
	switch {
	case step == string(FieldWeight):
		return "72,5"
	case Field(step).IsFold():
		return "12"
	case step == string(FieldHeight):
		return "175 ou 1,75"
	}
	return "85"
}

// check range-checks a numeric answer and either commits it, asks for it
// again, or parks it for a "sim".
func (s *Session) check(step string, v float64) Reply {
	f := Field(step)
	if step == string(FieldHeight) {
		f = FieldHeight
	}
	accept := func(c float64) bool {
		switch {
		case step == string(FieldWeight):
			return len(filterField(crossChecks(Input{HeightCM: s.HeightCM, WeightKG: c}), FieldWeight)) == 0
		case step == string(FieldHeight), f.IsFold():
			return true
		}
		return len(CheckRelation(f, c, s.HeightCM, s.Circumferences)) == 0
	}
	issues := validateValue(f, v, accept)
	if len(issues) == 0 {
		switch {
		case step == string(FieldWeight):
			issues = filterField(crossChecks(Input{HeightCM: s.HeightCM, WeightKG: v}), FieldWeight)
		case step == string(FieldHeight):
			if s.WeightKG > 0 {
				issues = filterField(crossChecks(Input{HeightCM: v, WeightKG: s.WeightKG}), FieldWeight)
			}
		case !f.IsFold():
			issues = CheckRelation(f, v, s.HeightCM, s.Circumferences)
		}
	}
	for _, i := range issues {
		switch {
		case i.Severity == SeverityError && i.Suggestion != nil:
			s.Pending = &Pending{Step: step, Value: *i.Suggestion}
			return s.reply(i.Message, step)
		case i.Severity == SeverityError:
			return s.reply(i.Message, step)
		}
	}
	for _, i := range issues { // only warnings are left
		s.Pending = &Pending{Step: step, Value: v}
		return s.reply(i.Message, step)
	}
	return s.commit(step, v)
}

func filterField(issues []Issue, f Field) []Issue {
	var out []Issue
	for _, i := range issues {
		if i.Field == f {
			out = append(out, i)
		}
	}
	return out
}

// reply repeats a refusal or a question without the tip, so a retry stays short.
func (s *Session) reply(msg, step string) Reply {
	return Reply{Text: msg, Step: step, Progress: s.progress()}
}

func (s *Session) commitSex(v Sex) Reply {
	s.Sex = v
	return s.advance(string(FieldSex), "")
}

func (s *Session) commitActivity(l ActivityLevel) Reply {
	s.Activity = l
	return s.advance(stepActivity, "Anotado: "+strings.ToLower(ActivityLabel(l))+".")
}

func (s *Session) commitGate(g string, yes bool) Reply {
	s.Gates[g] = yes
	return s.advance(g, "")
}

// commit stores a validated number for a step and moves on.
func (s *Session) commit(stepID string, v float64) Reply {
	var note string
	switch {
	case stepID == string(FieldAge):
		s.AgeYears = int(v)
		note = fmt.Sprintf("Anotado: %d anos.", int(v))
	case stepID == string(FieldHeight):
		s.HeightCM = v
		note = "Anotado: altura " + br(v) + " cm."
	case stepID == string(FieldWeight):
		s.WeightKG = v
		note = "Anotado: peso " + br(v) + " kg."
	case Field(stepID).IsFold():
		s.Skinfolds[Field(stepID)] = v
		note = "Anotado: " + labelOf(stepID) + " " + br(v) + " mm."
	default:
		s.Circumferences[Field(stepID)] = v
		note = "Anotado: " + labelOf(stepID) + " " + br(v) + " cm."
	}
	return s.advance(stepID, note)
}

func labelOf(stepID string) string {
	if spec, ok := SpecFor(Field(stepID)); ok {
		return spec.Label
	}
	return stepID
}

// advance records the step in the history and asks the next question, or
// finishes.
func (s *Session) advance(stepID, note string) Reply {
	s.History = append(s.History, stepID)
	s.Pending = nil
	s.Current = s.next()
	if s.Current != "" {
		return s.ask(note)
	}
	return s.finish(note)
}

// finish computes the estimate. A cross-field problem that only shows up now
// sends the person back to the value to blame instead of printing a number.
func (s *Session) finish(note string) Reply {
	res, err := Estimate(s.Input())
	var verr *ValidationError
	if errors.As(err, &verr) {
		first := verr.Issues[0]
		target := string(first.Field)
		if first.Field == "" || !s.answered(target) {
			target = string(Waist)
		}
		s.unset(target)
		s.Current = target
		return s.ask(first.Message)
	}
	s.Status = StatusDone
	s.Result = &res
	s.Current = ""
	text := FormatResult(res)
	if note != "" {
		text = note + "\n\n" + text
	}
	return Reply{Text: text, Done: true, Result: &res, Progress: s.progress()}
}

// unset forgets one step's answer.
func (s *Session) unset(stepID string) {
	switch stepID {
	case string(FieldSex):
		s.Sex = ""
	case string(FieldAge):
		s.AgeYears = 0
	case string(FieldHeight):
		s.HeightCM = 0
	case string(FieldWeight):
		s.WeightKG = 0
	case stepActivity:
		s.Activity = ""
	case stepMore, stepCaliper:
		delete(s.Gates, stepID)
	default:
		delete(s.Circumferences, Field(stepID))
		delete(s.Skinfolds, Field(stepID))
	}
	delete(s.Skipped, stepID)
	for i := len(s.History) - 1; i >= 0; i-- {
		if s.History[i] == stepID {
			s.History = append(s.History[:i], s.History[i+1:]...)
			break
		}
	}
}

// back undoes the last answer the person gave.
func (s *Session) back() Reply {
	s.Pending = nil
	if len(s.History) == 0 {
		return s.ask("Você já está na primeira pergunta.")
	}
	last := s.History[len(s.History)-1]
	s.unset(last)
	s.Current = last
	return s.ask("Voltei uma pergunta.")
}

// skip leaves an optional step blank.
func (s *Session) skip() Reply {
	if isGate(s.Current) {
		return s.ask("Responda *sim* ou *não*.")
	}
	if !s.optional(s.Current) {
		return s.ask("Essa medida é necessária para a estimativa, não dá para pular.")
	}
	s.Skipped[s.Current] = true
	return s.advance(s.Current, "Pulei essa.")
}

func isYes(w string) bool {
	switch w {
	case "sim", "s", "isso", "ok", "pode", "confirmo", "certo", "claro", "yes", "quero", "tenho":
		return true
	}
	return false
}

func isNo(w string) bool {
	switch w {
	case "nao", "n", "no", "nope", "nao quero", "agora nao", "nao tenho":
		return true
	}
	return false
}
