<p align="center">
  <img src=".github/logo.svg" alt="Arandu" width="180">
</p>

# Arandu Bodycomp

A digital bioimpedance for an [Arandu](https://github.com/arandu-io/framework)
application: it estimates body composition from what a person can measure at
home (height, weight, tape-measure circumferences and, optionally, caliper
skinfolds), compares two estimates over time, and drives the guided
conversation that collects the measurements.

Every number it returns is an **estimate, not a diagnosis**. It never reads a
photograph: a percentage comes only from the published formulas below, and every
result carries a disclaimer that says so.

It is a library. It registers no route, owns no table and decides no
authorization, so there is nothing to wire in `bootstrap/app.go`, nothing to
construct, and no `aru migrate` to run. It is the Go standard library alone:
the module has no dependency, does no I/O, holds no state and never reads the
clock (the caller passes "now" in). Storing measurements, photographs, consent
and who may see a result stay with the application that installs it.

## Install

```bash
go get github.com/hyz-is/arandu-bodycomp
```

```go
import bodycomp "github.com/hyz-is/arandu-bodycomp"
```

`arandu.mod.toml` declares no capability: no network, no filesystem, no
process, no migration.

## Use it

```go
// 1. Parse what a person types.
cm, err    := bodycomp.ParseLengthCM("1,75 m")      // 175
kg, err    := bodycomp.ParseMassKG("154 lb")        // 69.85
mm, err    := bodycomp.ParseSkinfoldMM("1,2 cm")    // 12
years, err := bodycomp.ParseAge("15/03/1990", now)  // completed years
sex, err   := bodycomp.ParseSex("mulher")

// 2. Validate: typos, impossible values, relations between values.
issues := bodycomp.Validate(input)
issues  = bodycomp.ValidateValue(bodycomp.Waist, 850) // suggests 85

// 3. Estimate.
res, err := bodycomp.Estimate(bodycomp.Input{
    Sex: bodycomp.Male, AgeYears: 30, HeightCM: 180, WeightKG: 80,
    Activity: bodycomp.Moderate,
    Circumferences: map[bodycomp.Field]float64{
        bodycomp.Neck: 40, bodycomp.Waist: 90, bodycomp.Hip: 100,
    },
})
// A *bodycomp.ValidationError means bad input: ask again instead of
// showing a number.
text := bodycomp.FormatResult(res) // a chat message, in Brazilian Portuguese

// 4. Evolution.
delta := bodycomp.Compare(previous, res, days)
lines := delta.Summary()

// 5. The guided conversation, a state machine that serialises to JSON.
s := bodycomp.NewSession(now, bodycomp.Prefill{Sex: bodycomp.Male, HeightCM: 180})
reply := s.Start()            // the first question
reply  = s.Answer("30", now)  // feed each message
reply  = s.Resume()           // after a pause, from stored JSON
// reply.Text is the message; reply.Done and reply.Result at the end;
// reply.Reason == "minor" when the person is under 18.

// 6. Texts for screens.
bodycomp.Methods() // an explanation and a citation for each method
bodycomp.Specs()   // label, range and "how to measure" tip for each measurement
```

During the conversation the person can send `ajuda`, `voltar`, `pular` and
`cancelar`, and answer `sim` or `não` to a confirmation. The conversation, the
validation messages and `FormatResult` are written in Brazilian Portuguese: they
are data this package ships, and the identifiers and comments of the code are in
English.

## What it computes

Each formula is one function with no validation and no rounding; `Estimate`
decides what is plausible. Lengths are centimetres, masses kilograms and
skinfolds millimetres.

| Output | Formula | Source |
| --- | --- | --- |
| BMI and band | kg/m², WHO adult bands | WHO Technical Report Series 894 (2000) |
| Waist to height | waist / height, reference 0.5 | Ashwell & Hsieh, Int J Food Sci Nutr 2005;56:303 |
| Waist to hip | waist / hip, WHO limits 0.90 (men) and 0.85 (women) | WHO expert consultation, Geneva 2008 |
| Body fat, U.S. Navy | 495 / (1.0324 − 0.19077 log10(waist − neck) + 0.15456 log10(height)) − 450 (men); 495 / (1.29579 − 0.35004 log10(waist + hip − neck) + 0.22100 log10(height)) − 450 (women) | Hodgdon & Beckett, NHRC reports 84-11 and 84-29 (1984); Siri 1961 |
| Body fat, RFM | 64 − 20 height / waist (+12 for women) | Woolcott & Bergman, Sci Rep 2018;8:10980 |
| Body fat, CUN-BAE | from BMI, age and sex (nine terms) | Gómez-Ambrosi et al., Diabetes Care 2012;35:383 |
| Body fat, skinfolds | Jackson and Pollock, three sites, then Siri | Jackson & Pollock, Br J Nutr 1978;40:497; Jackson, Pollock & Ward, Med Sci Sports Exerc 1980;12:175 |
| Fat and lean mass | weight × the consensus body fat | |
| FFMI and normalised FFMI | lean mass / height²; plus 6.1 (1.8 − height) | Kouri et al., Clin J Sport Med 1995;5:223 |
| Resting energy | Mifflin-St Jeor; Katch-McArdle | Mifflin et al., Am J Clin Nutr 1990;51:241; McArdle, Katch & Katch |
| Daily energy | resting × 1.2, 1.375, 1.55, 1.725, 1.9 | Harris-Benedict tradition, as in ACSM and McArdle texts |
| Body fat band | essential, athlete, fitness, average | American Council on Exercise bands |

The consensus body fat is the median of the methods that ran (the mean of two).
A method whose output falls outside 2 to 65 percent is dropped with a warning
instead of being averaged in. `Agreement` says how close the methods are: a
spread up to 3 points is high, up to 6 medium, above that low. A body-fat change
smaller than 1.5 points between two dates is reported as no clear change,
because the formulas' own error is larger than that.

## Rules the code enforces

- **Adults only**, 18 to 90. A minor ends the conversation, and nothing is kept.
- **Out-of-range values are refused.** A likely slip (a lost decimal point,
  pounds typed as kilograms, inches as centimetres) is offered as a correction
  the person confirms. Relations are checked at entry: the waist larger than
  the neck, the flexed arm not smaller than the relaxed one, a plausible BMI.
- **Units**: cm, m, mm, in, ft+in, kg, lb and g, with Brazilian decimal commas
  and common misspellings (`cms`, `quilos`, `7O` for 70).
- **An estimate, never a diagnosis.** The disclaimer is always set on a result,
  and the package never says a person has or lacks a condition.

## What the application keeps

The package holds nothing, so an application that installs it decides all of
this, under its own Arandu modules:

- where an estimate and a session are stored (`Session` and `Result` serialise
  to JSON);
- who may read them: through its own policies, Grants and tenant, with
  `hyz-is/arandu-permission` for roles;
- the consent a person gives before measurements are kept, and photographs, if
  there are any;
- the log of who read what, with `hyz-is/arandu-activitylog`.

Body measurements are personal data about health. Treat them as sensitive.

## Tests

```bash
export GOWORK=off
go test -race ./...
```

The formula tests check values computed independently (in double precision)
from the printed equations, plus the equality of the metric Navy forms with the
inch forms printed in the Navy regulation. `tests/Unit` calls the package from
outside, holds the library shape (standard library only, no route, no table, no
`init`, no clock) and compares `arandu.mod.toml` with what the code calls.

## Licence

MIT. See [LICENSE.md](LICENSE.md).
