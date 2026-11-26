# Contributing

Thank you for considering a contribution.

## Before you open a pull request

Run the four gates. They are what CI runs, and a pull request that fails one of
them fails for everybody who pulls after it.

```bash
gofmt -l $(find . -name '*.go' -not -path '*/testdata/*' -not -name '*.kyse.go')
go build ./...
go vet ./...
go test -race ./...
```

The two filters on `gofmt` are not preference. `gofmt` is the only tool in the
chain that ignores build tags, so it reads files the compiler deliberately
excludes; `testdata/` holds fixtures that are invalid on purpose.

## What a change has to keep true

1. **A formula is a published formula.** Each function in `formulas.go` is the
   printed equation, with its source in the doc comment, and its test compares
   it with a value computed apart from the package. A change to a coefficient
   needs the publication that prints it; it is never tuned to make a result look
   better.
2. **An estimate, never a diagnosis.** `Estimate` always sets `Disclaimer`, and
   no text the package writes says that a person has or lacks a condition.
3. **A minor is refused, and nothing is kept.**
4. **This stays a library.** No route, no table, no policy, no `Module`, no
   `init`, no clock, and no import beyond the standard library.
   `TestThePackageIsALibrary` holds all of it.
5. **`arandu.mod.toml` matches the code.** Adding an outbound call, a file
   access or a process means declaring it there in the same commit.

No real person's measurements go into a test, a comment or a document.

## Style

Everything in the source is in English: identifiers, doc comments, internal
comments, error messages, log messages, and the names and messages of tests.
`pkg.go.dev` publishes the doc comments, and its readers are users of this
package. The text a person reads in a chat (the questions of the conversation,
the validation messages, `FormatResult`, the method explanations) is data the
package ships, in Brazilian Portuguese, and stays so.

Every exported symbol carries a doc comment saying what it does. The comment
documents the symbol and nothing else: no dates, no issue numbers, no
references to other repositories. Why a signature is what it is belongs there
when it is a fact about the code; anything else belongs in the pull request.

## Commits

One commit per logical change. Formatting is not mixed with behaviour, because
a diff that does both is a diff nobody reviews.

## Reporting a vulnerability

Do not open an issue. See [SECURITY.md](SECURITY.md).

## Licence

By contributing you agree that your contribution is licensed under the MIT
licence, the same as the rest of github.com/hyz-is/arandu-bodycomp.
