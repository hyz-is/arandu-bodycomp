package bodycomp

// MethodInfo explains one formula to the person reading a result: what it
// uses, what it is good and bad at, and where it comes from. The text is
// pt-BR, plain, and the single source for every screen and message that
// explains a method.
type MethodInfo struct {
	Method Method
	Name   string
	Needs  string
	// Plain is what the method does, in a sentence a layperson follows.
	Plain string
	// Limits is where it fails.
	Limits string
	// Citation is the publication the equation comes from.
	Citation string
}

var methodInfos = []MethodInfo{
	{
		Method:   MethodNavy,
		Name:     "Marinha dos EUA",
		Needs:    "altura, cintura e pescoço (mulheres: e quadril)",
		Plain:    "Compara o tamanho da cintura com o do pescoço e a altura. Quanto mais a cintura passa do pescoço, maior a gordura estimada.",
		Limits:   "Costuma errar alguns pontos percentuais. Erra mais em quem tem muito músculo no pescoço ou gordura concentrada fora da barriga.",
		Citation: "Hodgdon e Beckett, Naval Health Research Center, relatórios 84-11 (homens) e 84-29 (mulheres), 1984.",
	},
	{
		Method:   MethodRFM,
		Name:     "Gordura relativa (RFM)",
		Needs:    "altura e cintura",
		Plain:    "Usa só a razão entre a altura e a cintura. É simples e foi ajustada com exames de DXA em milhares de pessoas.",
		Limits:   "Ignora peso, idade e músculo, então tende a errar em atletas e em pessoas muito magras.",
		Citation: "Woolcott e Bergman, Scientific Reports 2018;8:10980.",
	},
	{
		Method:   MethodCUNBAE,
		Name:     "CUN-BAE",
		Needs:    "peso, altura, idade e sexo",
		Plain:    "Estima a gordura pelo IMC, corrigindo por idade e sexo. Não usa nenhuma fita métrica.",
		Limits:   "Não enxerga músculo: quem treina muito aparece com mais gordura do que tem. Serve de conferência para os outros métodos.",
		Citation: "Gómez-Ambrosi e cols., Diabetes Care 2012;35:383-388.",
	},
	{
		Method:   MethodJacksonP3,
		Name:     "Dobras cutâneas (Jackson e Pollock, 3 pontos)",
		Needs:    "três dobras medidas com adipômetro, idade e sexo",
		Plain:    "Soma a espessura de três dobras de pele e converte em densidade corporal e gordura (equação de Siri).",
		Limits:   "Depende de quem mede: a mesma pessoa, sempre com o mesmo adipômetro, dá a comparação mais confiável.",
		Citation: "Jackson e Pollock, British Journal of Nutrition 1978;40:497-504 (homens); Jackson, Pollock e Ward, Medicine and Science in Sports and Exercise 1980;12:175-181 (mulheres); Siri, 1961.",
	},
}

// Methods lists the explanations of every body-fat method, in a stable order.
// The slice is a copy.
func Methods() []MethodInfo { return append([]MethodInfo(nil), methodInfos...) }

// Info returns the explanation of one method.
func Info(m Method) (MethodInfo, bool) {
	for _, i := range methodInfos {
		if i.Method == m {
			return i, true
		}
	}
	return MethodInfo{}, false
}

// MethodName returns the short pt-BR name, or the identifier for an unknown
// method.
func MethodName(m Method) string {
	if i, ok := Info(m); ok {
		return i.Name
	}
	return string(m)
}

// AgreementText explains a result's agreement level in one sentence.
func AgreementText(a Agreement, spreadPP float64) string {
	switch a {
	case AgreementHigh:
		return "Os métodos concordam entre si (diferença de " + br(spreadPP) + " pontos). A estimativa é mais confiável."
	case AgreementMedium:
		return "Os métodos diferem em " + br(spreadPP) + " pontos. Use o valor central e olhe a evolução ao longo do tempo."
	case AgreementLow:
		return "Os métodos diferem em " + br(spreadPP) + " pontos, bastante. Confira as medidas e prefira comparar a evolução em vez do valor isolado."
	}
	return "Só um método pôde ser calculado com as medidas informadas. Mais medidas melhoram a estimativa."
}
