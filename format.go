package bodycomp

import (
	"fmt"
	"strings"
)

// FormatResult writes a result as a chat message in pt-BR: the answer first,
// then the numbers a person acts on, then the caveat. WhatsApp bold is *text*.
func FormatResult(r Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "*Sua estimativa de composição corporal*\n")
	fmt.Fprintf(&b, "Gordura corporal: *%s%%* (%s)\n", br(r.BodyFatPercent), r.FatCategory)
	fmt.Fprintf(&b, "Massa gorda: %s kg · Massa magra: %s kg\n", br(r.FatMassKG), br(r.LeanMassKG))
	fmt.Fprintf(&b, "IMC: %s (%s)\n", br(r.BMI), r.BMIClass)
	if r.WaistToHeight != nil {
		fmt.Fprintf(&b, "Cintura/altura: %s (%s)\n", br2(*r.WaistToHeight), r.WaistToHeightNote)
	}
	if r.WaistToHip != nil {
		fmt.Fprintf(&b, "Cintura/quadril: %s (%s)\n", br2(*r.WaistToHip), r.WaistToHipNote)
	}
	fmt.Fprintf(&b, "Índice de massa magra (FFMI): %s\n", br(r.FFMI))
	fmt.Fprintf(&b, "Metabolismo basal: cerca de %d kcal por dia", int(r.Energy.BMRMifflin))
	if r.Energy.TDEE > 0 {
		fmt.Fprintf(&b, "; gasto total no seu nível de atividade: cerca de %d kcal", int(r.Energy.TDEE))
	}
	b.WriteString("\n\n")
	if len(r.Methods) > 1 {
		b.WriteString("Por método: ")
		parts := make([]string, 0, len(r.Methods))
		for _, m := range r.Methods {
			parts = append(parts, fmt.Sprintf("%s %s%%", MethodName(m.Method), br(m.Percent)))
		}
		b.WriteString(strings.Join(parts, "; "))
		b.WriteString(".\n")
	}
	b.WriteString(AgreementText(r.Agreement, r.SpreadPP))
	b.WriteString("\n\n_")
	b.WriteString(r.Disclaimer)
	b.WriteString("_")
	return b.String()
}
