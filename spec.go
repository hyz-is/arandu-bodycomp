package bodycomp

// Spec describes one measurement: how to name it to a person, where to place
// the tape, what range is humanly possible and in which unit it is stored.
//
// The ranges are deliberately wide. They exist to catch a typo ("850" for
// 85 cm, a weight typed in pounds without the unit), not to judge a body. A
// person at the edge of a range is asked once to confirm; a value outside it
// is asked again.
type Spec struct {
	Field Field
	Unit  Unit
	// Label is the short pt-BR name used in questions.
	Label string
	// Min and Max bound a plausible adult value, in Unit.
	Min, Max float64
	// tipMale and tipFemale are the "como medir" lines. An empty tipFemale
	// means the male text applies to both.
	tipMale, tipFemale string
}

// Tip returns the "como medir" line for the person's sex. An unknown sex gets
// the shared text.
func (s Spec) Tip(sex Sex) string {
	if sex == Female && s.tipFemale != "" {
		return s.tipFemale
	}
	return s.tipMale
}

var specs = []Spec{
	{Field: FieldHeight, Unit: UnitCM, Label: "altura", Min: 120, Max: 230,
		tipMale: "Descalço, de costas para a parede, olhando para frente. Marque o topo da cabeça com um livro reto e meça do chão até a marca."},
	{Field: FieldWeight, Unit: UnitKG, Label: "peso", Min: 30, Max: 300,
		tipMale: "Pese-se de manhã, em jejum, depois de ir ao banheiro, com pouca roupa. Sempre na mesma balança."},

	{Field: Neck, Unit: UnitCM, Label: "pescoço", Min: 25, Max: 60,
		tipMale: "Fita logo abaixo do gogó (pomo de Adão), reta em volta do pescoço, olhando para frente, sem encolher os ombros nem apertar a fita."},
	{Field: Waist, Unit: UnitCM, Label: "cintura", Min: 40, Max: 200,
		tipMale:   "Fita na altura do umbigo, paralela ao chão. Solte o ar, relaxe a barriga e leia sem puxar a fita.",
		tipFemale: "Fita no ponto mais estreito da cintura, acima do umbigo (se não achar o mais estreito, use a altura do umbigo). Paralela ao chão, depois de soltar o ar."},
	{Field: Hip, Unit: UnitCM, Label: "quadril", Min: 60, Max: 200,
		tipMale: "Fita no ponto mais largo do bumbum, paralela ao chão, pés juntos, sem apertar."},
	{Field: Abdomen, Unit: UnitCM, Label: "abdômen", Min: 40, Max: 200,
		tipMale: "Fita no ponto mais saliente da barriga, normalmente 2 a 4 cm abaixo do umbigo, depois de soltar o ar."},
	{Field: Chest, Unit: UnitCM, Label: "tórax", Min: 60, Max: 180,
		tipMale: "Fita em volta do peito, na altura dos mamilos, por baixo dos braços e reta nas costas. Meça com o ar solto."},
	{Field: Shoulders, Unit: UnitCM, Label: "ombros", Min: 80, Max: 170,
		tipMale: "Fita em volta dos ombros, passando pela parte mais larga do deltoide e pelo meio do peito. Peça ajuda para manter a fita reta."},
	{Field: ArmRelaxed, Unit: UnitCM, Label: "braço relaxado", Min: 15, Max: 60,
		tipMale: "Braço solto ao lado do corpo. Fita no meio entre o ombro e o cotovelo. Use sempre o mesmo braço."},
	{Field: ArmFlexed, Unit: UnitCM, Label: "braço contraído", Min: 15, Max: 65,
		tipMale: "Braço levantado na frente, cotovelo dobrado, bíceps contraído com força. Fita no ponto mais alto do músculo."},
	{Field: Forearm, Unit: UnitCM, Label: "antebraço", Min: 15, Max: 45,
		tipMale: "Braço estendido, palma para cima. Fita no ponto mais grosso do antebraço, perto do cotovelo."},
	{Field: Thigh, Unit: UnitCM, Label: "coxa", Min: 25, Max: 100,
		tipMale: "Em pé, peso dividido nas duas pernas. Fita no meio entre a dobra da virilha e o joelho. Use sempre a mesma perna."},
	{Field: Calf, Unit: UnitCM, Label: "panturrilha", Min: 20, Max: 60,
		tipMale: "Em pé, peso dividido nas duas pernas. Fita no ponto mais grosso da panturrilha."},

	{Field: FoldChest, Unit: UnitMM, Label: "dobra do peito", Min: 2, Max: 60,
		tipMale: "Dobra diagonal, na metade do caminho entre a axila e o mamilo. Puxe a pele com dois dedos e leia o adipômetro após 2 segundos."},
	{Field: FoldAbdomen, Unit: UnitMM, Label: "dobra do abdômen", Min: 3, Max: 60,
		tipMale: "Dobra vertical, 2 cm ao lado do umbigo. Puxe a pele com dois dedos e leia após 2 segundos."},
	{Field: FoldThigh, Unit: UnitMM, Label: "dobra da coxa", Min: 3, Max: 60,
		tipMale: "Dobra vertical, no meio da frente da coxa, entre a virilha e o joelho. Perna relaxada."},
	{Field: FoldTriceps, Unit: UnitMM, Label: "dobra do tríceps", Min: 3, Max: 60,
		tipMale: "Dobra vertical, atrás do braço, no meio entre o ombro e o cotovelo. Braço solto."},
	{Field: FoldSuprailiac, Unit: UnitMM, Label: "dobra suprailíaca", Min: 3, Max: 60,
		tipMale: "Dobra diagonal, logo acima do osso do quadril, na linha do meio da axila. Puxe a pele no sentido da dobra."},
}

var specIndex = func() map[Field]Spec {
	m := make(map[Field]Spec, len(specs))
	for _, s := range specs {
		m[s.Field] = s
	}
	return m
}()

// SpecFor returns the specification of a measurement field. The second result
// is false for a field the package does not measure (sex and age have no Spec).
func SpecFor(f Field) (Spec, bool) {
	s, ok := specIndex[f]
	return s, ok
}

// Specs returns every measurement specification in a stable order. The slice
// is a copy.
func Specs() []Spec { return append([]Spec(nil), specs...) }

// IsFold reports whether f is a skinfold.
func (f Field) IsFold() bool {
	s, ok := specIndex[f]
	return ok && s.Unit == UnitMM
}

// FoldsFor returns the three skinfold sites of the Jackson-Pollock three-site
// equation for a sex: chest, abdomen and thigh for men; triceps, suprailiac
// and thigh for women.
func FoldsFor(sex Sex) []Field {
	if sex == Female {
		return []Field{FoldTriceps, FoldSuprailiac, FoldThigh}
	}
	return []Field{FoldChest, FoldAbdomen, FoldThigh}
}
