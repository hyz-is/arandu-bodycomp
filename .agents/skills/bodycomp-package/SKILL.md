---
name: bodycomp-package
description: Estimate body composition in an Arandu application with the Arandu Bodycomp package (Go, github.com/hyz-is/arandu-bodycomp). Use when the request is to "estimate body fat", "body composition", "digital bioimpedance", "bioimpedancia digital", "BMI and waist ratios", "Navy body fat", "RFM", "CUN-BAE", "skinfolds", "Jackson Pollock", "FFMI", "resting energy", "TDEE", "compare two measurements", "guided measurement conversation", "parse 1,75 m or 154 lb", "go get github.com/hyz-is/arandu-bodycomp", or when the application has its own body-fat, BMI, TMB or Mifflin helper, or when a project's go.mod already requires github.com/hyz-is/arandu-bodycomp. Covers what the package computes, the call order, errors that mean ask again, storing a Result and a Session, passing the clock in, the disclaimer, and what stays with the application.
license: MIT
metadata:
  audience: app
---

# Using Arandu Bodycomp

A library that estimates body composition from self-measurements with published
formulas. It has no route, no table and no policy, so nothing about it goes in
`bootstrap/app.go`: there is no `New`, no line in `k.Register` and no
`aru migrate`. Import it and call it.

## Install

```bash
go get github.com/hyz-is/arandu-bodycomp
```

```go
import bodycomp "github.com/hyz-is/arandu-bodycomp"
```

## If the application already does this itself

Replace it; do not wrap it. A second set of coefficients in the application is
the one that drifts from the publication.

| the application has | it becomes |
| --- | --- |
| a BMI or a waist-to-height helper | `bodycomp.BMI`, `bodycomp.WaistToHeight`, or the fields of `Result` |
| a Navy, RFM or skinfold body-fat formula | `bodycomp.Estimate`, which runs every method the data allows and returns the consensus |
| a Mifflin or Katch-McArdle energy helper | `Result.Energy`, or `bodycomp.MifflinStJeor` and `bodycomp.KatchMcArdle` |
| a parser of "1,75 m", "70 kg", "154 lb" | `bodycomp.ParseLengthCM`, `ParseMassKG`, `ParseSkinfoldMM` |
| a before-and-after of two measurements | `bodycomp.Compare` and `Delta.Summary` |
| a hand-written question-by-question flow | `bodycomp.Session` |

## The call order

```go
issues := bodycomp.Validate(input)          // optional: the same checks, as a list
res, err := bodycomp.Estimate(input)
var invalid *bodycomp.ValidationError
if errors.As(err, &invalid) {
    // Bad input. Ask again with invalid.Issues; never show a number.
}
text := bodycomp.FormatResult(res)          // pt-BR chat message, with the disclaimer
delta := bodycomp.Compare(previous, res, days)
```

## Rules to keep when calling it

- **Pass the clock in.** `NewSession(now, ...)`, `Answer(text, now)` and
  `ParseAge(text, now)` take the time; the package never reads it. The caller
  decides what "now" is, which is also what makes a test reproducible.
- **Show the disclaimer.** `Result.Disclaimer` is always set. A screen or a
  message that shows an estimate shows it, and never writes that a person has
  or lacks a condition.
- **Ask again on a `*ValidationError`.** It is not a failure to log; it is the
  next question. A likely slip (a lost decimal point, pounds typed as
  kilograms) comes back as a correction the person confirms.
- **A minor ends it.** `Reply.Reason == bodycomp.ReasonMinor` means the person is
  under 18: show the refusal and keep nothing.
- **Store a `Result` and a `Session` as JSON,** and read them back with
  `json.Unmarshal`. A `Session` saved after each `Answer` lets a person stop and
  `Resume()` days later.

## What stays with the application

The package stores nothing and authorizes nothing, so the application decides,
with its own Arandu modules:

- the table, owned by an Organization, written with the Grant's tenant
  (`data.Tenant(g)`), never a tenant from the request;
- who may read a result: its own Policy, with `hyz-is/arandu-permission` for roles;
- the consent a person gives before measurements are kept;
- the log of who read what, with `hyz-is/arandu-activitylog`.

Body measurements are health data. Treat them as sensitive: never in a log line,
a URL or a public bucket.

## Symptoms

| you see | it means |
| --- | --- |
| `*bodycomp.ValidationError` from `Estimate` | an error-level issue in the input; read `Issues` and ask again |
| `Reply.Cancelled` with `Reason == "minor"` | the person is under 18; the session kept nothing |
| a body fat that differs from another site's | each method has a published error of a few points; `Result.Agreement` says how close the methods are |
| `Delta.FatWithinNoise` is true | the change is smaller than the formulas' own error; say so, do not celebrate it |
