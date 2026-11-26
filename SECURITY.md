# Security policy

## Supported versions

The latest minor release of Arandu Bodycomp receives security fixes. Older minor
releases do not: a published Go module version is immutable, so a fix is a new
release rather than a moved tag.

## Reporting a vulnerability

Report it privately, through GitHub's advisory form:

<https://github.com/hyz-is/arandu-bodycomp/security/advisories/new>

Do not open a public issue and do not describe the problem in a pull request.
A report that arrives in public is a report every reader of this repository can
act on before there is a release to upgrade to.

Include what you need to reproduce it: the version and the input that triggers
it. Never a real person's measurements: an invented set shows the same defect.

Expect an acknowledgement within a few days. If the report is confirmed, the
fix, the release and the advisory are published together, and you are credited
unless you ask not to be.

## What is in scope

Anything that makes the package answer in a way its documentation says it does
not. In particular:

- a panic, or an endless loop, on any input to any exported function or to a
  `Session`, including one restored from stored JSON;
- a minor (under 18) that `Estimate`, `Validate` or the conversation accepts, or
  whose answers a `Session` keeps after refusing;
- a `Result` without its `Disclaimer`, or wording that states a diagnosis;
- a formula that differs from the equation printed in the source it cites;
- an import outside the standard library, or a call that leaves the process,
  touches a file or runs a program.

## What is not

- An estimate that is far from a person's real body composition. Every number
  is an estimate from a published formula with a published error, and the
  package says so; it is not a medical device and it is not a diagnosis.
- How an application stores, shows or shares a result. That is the
  application's, under its own policies.
