---
name: bodycomp-release
description: Get this Arandu package green, declared and published. Use when the request is to "run the gates", "check it builds", "the tests fail", "go build says the directory prefix does not contain modules listed in go.work", "aru doctor says this is not an Arandu project", "update arandu.mod.toml", "add a dependency", "change a formula", "tag a release", "publish it", "go get cannot find the module", "write the changelog", or before opening a pull request. Covers the four gates, what the manifest declares and who checks it, why the module requires nothing, the immutability of a published module version, and what breaks somebody else's numbers without touching a signature.
license: MIT
---

# Getting it green and out

## The four gates

Nothing is finished until all four exit zero.

```sh
export GOWORK=off
gofmt -l $(find . -name '*.go' -not -path '*/testdata/*' -not -name '*.kyse.go')
go build ./...
go vet ./...
go test -race ./...
```

`GOWORK=off` first, and here it is not a habit. This checkout may sit beside a
Go workspace that lists the framework repositories and does not list this one.
When it does, `go build ./...` exits 1 before compiling anything:

```
pattern ./...: directory prefix . does not contain modules listed in go.work
or their selected dependencies
```

That is the workspace refusing to answer for a module it was not told about, and
it says nothing about the code.

`aru doctor` is not a gate. It exits 1 here with *"this is not an Arandu
project: no go.mod, main.go and arandu.toml together"*: it reads applications,
and this is a library.

## What the manifest declares, and who reads it

`arandu.mod.toml` is a promise, and the check that gives it weight runs here.
```toml
name = "hyz-is/bodycomp"
framework = ">= 0.71"
profiles = ["conventional", "performance"]

[permissions]
network = false
filesystem = false
exec = false
migrations = false
```

`aru doctor` audits the application it is run inside and never loads a
dependency, so the manifest of an installed package is opened by nothing but
that package's own suite.

`TestTheDeclaredCapabilitiesAreWhatTheCodeDoes`, in `tests/Unit/audit_test.go`,
compares the four permissions against what the code *calls*, by AST. All four
are `false`, and they stay that way: the package turns numbers and text into
numbers and text. A call that stores, fetches or runs something would make it a
different package, not a change to this one.

The `framework` line is a floor for the applications that install the package.
The module imports no framework, so nothing in `go.mod` moves it; raise it
only when the applications this package is installed in need a newer one.

## Adding a dependency

Do not. This module goes into other people's builds and requires nothing.
`TestTheModuleRequiresNothing` refuses any `require` in `go.mod`, and
`TestThePackageIsALibrary` refuses an import beyond the standard library, and
`os`, `net/http` and `os/exec` within it.

## Changing a formula

A formula is a published equation. Before touching `formulas.go`:

- find the publication that prints the change, and put it in the doc comment;
- compute the expected values apart from the package (double precision, from
  the printed equation) and put them in the test;
- never tune a coefficient so that a result looks better.

A changed formula changes every stored comparison: `Compare` of an old and a new
estimate would show a change the person's body did not make. Say so in the
changelog and in `UPGRADE.md`.

## Publishing

A published Go module version is **immutable**. The proxy serves it forever, and
a mistake is corrected by another release, never by moving a tag.

Before tagging:

- the four gates pass with `GOWORK=off`;
- `arandu.mod.toml` matches the code;
- `CHANGELOG.md` has the entry under the version, in the Keep a Changelog
  sections (`Added`, `Changed`, `Fixed`, `Removed`), and `UPGRADE.md` has a
  `## vX.Y.Z` heading for the same version;
- `currentVersion` in `tests/Unit/release_test.go` names that version.

While the version starts with `v0.`, breaking is allowed. Breaking *quietly* is
not. The CI step named `api diff against the last release` fails unless
`UPGRADE.md` names each removed or changed symbol.

Three changes break an installer without touching a signature, and each one is
worth a line in the changelog and in `UPGRADE.md`:

- **A formula or a coefficient.** The same input gives another number, and
  every stored comparison moves.
- **A threshold.** The plausible range of a method, `FatNoisePP`, the agreement
  spread: they decide what is shown, what is dropped and what is called a change.
- **A conversation step or its wording.** A stored `Session` resumes at a step
  by name; renaming one strands every person halfway through.

## Reporting a vulnerability

Not in an issue and not in a pull request. `SECURITY.md` has the private
advisory address. A panic on any input, a minor accepted, or a result without
its disclaimer is in scope.
