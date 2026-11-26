# Changelog

Everything worth knowing about a release of Arandu Bodycomp is recorded here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
the versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

A published module version is immutable: Go serves it from the proxy forever, so
a release is corrected by another release and never by moving a tag.

## [Unreleased]

## [0.1.0] - 2026-11-25

### Added

- `Estimate`: body composition from height, weight, circumferences and
  optional skinfolds. It returns BMI and its band, waist to height, waist to
  hip, body fat by the U.S. Navy method, RFM, CUN-BAE and Jackson-Pollock
  three-site skinfolds, the consensus (the median of the methods that ran) with
  its `Agreement`, fat and lean mass, FFMI and normalised FFMI, resting energy
  by Mifflin-St Jeor and Katch-McArdle, and daily energy at five activity
  levels. A method outside 2 to 65 percent is dropped with a warning, and every
  `Result` carries its `Disclaimer`: an estimate, not a diagnosis.
- The formulas as functions, one each and with their sources: `BMI`,
  `WaistToHeight`, `WaistToHip`, `NavyMale`, `NavyFemale`, `RFM`, `CUNBAE`,
  `JacksonPollock3`, `FFMI`, `NormalizedFFMI`, `MifflinStJeor` and
  `KatchMcArdle`.
- `Compare` and `Delta.Summary`: the change between two estimates, with a body
  fat change under `FatNoisePP` reported as no clear change.
- `ParseLengthCM`, `ParseMassKG`, `ParseSkinfoldMM`, `ParseAge` and `ParseSex`:
  what a person types, with Brazilian decimal commas and common misspellings.
- `Validate` and `ValidateValue`: range checks, likely slips offered as a
  correction, and relations between values. Adults only, 18 to 90.
- `Session`, `Prefill` and `Reply`: the guided conversation as a state machine
  that serialises to JSON, with `ajuda`, `voltar`, `pular` and `cancelar`. A
  minor ends it with `Reason == "minor"` and nothing kept.
- `FormatResult`, `Methods`, `Info` and `Specs`: the chat message and the texts
  for screens, in Brazilian Portuguese.
- The module has no dependency, and `arandu.mod.toml` declares no capability.

