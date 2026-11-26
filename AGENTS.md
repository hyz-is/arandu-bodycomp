# Working on Arandu Bodycomp

This is an Arandu package that estimates body composition: BMI, waist ratios,
body fat by four published methods, fat and lean mass, FFMI and energy
expenditure, from measurements a person takes at home. It compares two
estimates over time and runs the guided conversation that collects the
measurements. It is a Go module somebody `go get`s and imports.

It began as the package skeleton, configured as `hyz-is/bodycomp`, and the HTTP
module the template ships was removed on purpose: there is no entity, no table,
no migration, no policy, no route, no view and no `Module` to register, and
nothing here should grow back into it. The module requires nothing: not the
framework, not Hesape, not a third-party package.

Read `.agents/skills/` before writing code. Each skill is a procedure, and the
one you need is named by the situation you are in.

## The gates

Nothing is finished until all four exit zero.

```sh
export GOWORK=off
gofmt -l $(find . -name '*.go' -not -path '*/testdata/*' -not -name '*.kyse.go')
go build ./...
go vet ./...
go test -race ./...
```

`GOWORK=off` is not a preference. This checkout may sit beside a Go workspace
that lists the framework repositories and does not list this one; when it does,
every command above fails before it compiles anything:

```
pattern ./...: directory prefix . does not contain modules listed in go.work
or their selected dependencies
```

`aru doctor` is not one of the gates. It reads applications, and exits 1 here.
It does not read this package after it is installed either, which is why
`tests/Unit/audit_test.go` compares `arandu.mod.toml` with what the code calls.

## What this repository holds

| | measured with |
| --- | --- |
| 10 Go files in one package at the root | `grep -l '^package bodycomp' *.go` |
| 6 internal test files beside the code, 4 files under `tests/Unit` | `find . -name '*_test.go'` |
| 0 routes, 0 tables, 0 capabilities | `arandu.mod.toml`, `TestThePackageIsALibrary` |
| 0 dependencies | `go list -m all` prints the module alone |

```
types.go      the package comment, Sex, Field, Input, Method, Issue and ValidationError
units.go      parsing what a person types: lengths, masses, skinfolds, age, sex
validate.go   range checks, likely slips and relations between values
formulas.go   one function per published formula, no validation and no rounding
estimate.go   Estimate: the consensus, the agreement, the bands and the energy
compare.go    Compare and Delta.Summary
flow.go       Session, the serialisable conversation
format.go     FormatResult, the chat message
methods.go    Methods and Info: the explanation and the citation of each method
spec.go       Specs: label, range and the "how to measure" tip of each measurement
```

## What does not exist here, on purpose

| A model reaches for | What is here instead |
| --- | --- |
| `New`, a `Module`, a `Config`, a line in `k.Register` | nothing. A caller imports the package and calls a function |
| a table or a repository for measurements | nothing. The application stores `Result` and `Session`, which serialise to JSON, under its own policies and tenant |
| `time.Now()` | a `now` argument. The same input gives the same answer on any day; `TestTheEngineKeepsNoGlobalClock` holds it |
| an `init`, a global registry, a package-level variable that changes | nothing. Importing the package does nothing |
| an import outside the standard library, `os`, `net/http` or `os/exec` | nothing. `TestThePackageIsALibrary` holds the list |
| a call to a service that reads a photograph or a body scan | nothing. A percentage comes only from a formula; a photograph is not an input |
| wording that says a person has or lacks a condition | an estimate with its disclaimer. Every `Result` carries `Disclaimer` |

## The four properties

A change that breaks one of them is not merged, whatever else it improves.

1. **A formula is a published formula.** Each function in `formulas.go` is the
   printed equation, with its source in the doc comment, and its test compares
   it with a value computed apart from the package. A coefficient is never
   tuned to make a result look better.
2. **An estimate, never a diagnosis.** `Estimate` always sets `Disclaimer`, a
   method outside its plausible range is dropped with a warning rather than
   averaged in, and a change smaller than the formulas' own error is reported
   as no clear change.
3. **A minor is refused, and nothing is kept.** Adults only, 18 to 90, in the
   validation and in the conversation.
4. **`arandu.mod.toml` matches the code.** All four capabilities are `false`,
   and `tests/Unit/audit_test.go` compares that with what the code calls.

## Writing code

Everything in the source is in English: identifiers, doc comments, internal
comments, error messages, and the names and messages of tests. The text a
person reads in a chat (the questions of the conversation, the validation
messages, `FormatResult`, the method explanations) is data the package ships, in
Brazilian Portuguese, and stays so.

Every exported symbol carries a doc comment, and the comment documents the
symbol and nothing else. Why a signature is what it is belongs there when it is
a fact about the code. A date, an issue number, a version in progress or the
name of another repository does not.

Tests go under `tests/Unit`, declaring `package unit_test`, and import the
package by its module path, which is what makes them see exactly what a caller
sees. A test that genuinely needs something unexported goes beside the code as
`*_internal_test.go`, and the suffix is how it says so.
