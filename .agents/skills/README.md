# Skills

Procedures an assistant follows when working on this package.

They live in `.agents/skills/<name>/SKILL.md`, which is the path the coding
assistants read from: Cursor, Codex, Cline, Copilot, Gemini CLI, Amp, OpenCode,
Warp, Zed and the rest all look there. It is one directory rather than a file
per vendor, so a skill written once is read by whatever this package is being
written with.

Each file opens with frontmatter carrying a `name` and a `description`. The
`name` has to equal the directory name exactly, or the skill is not loaded. The
description is what a tool reads to decide whether the skill is relevant, so it
names the situation you are in rather than the topic it covers.

| skill | when it fires |
| --- | --- |
| `bodycomp-release` | the gates, the manifest, a dependency, a formula change, a version, a tag |
| `bodycomp-package` | estimating body composition, comparing two measurements or running the guided conversation **in an application** |

The template this package started from carried a `-policy`, a `-module` and a
`-vault-notes` skill. The first two taught the HTTP module the template ships,
and this package removed it: there is no policy to open and no route, handler
or migration to add.

`bodycomp-package` has a different audience from the other one, and that is on
purpose: it travels with the package so that an assistant working in somebody
else's project, the one running `go get`, has the call order, the errors that
mean "ask again" and the limits of an estimate in front of it instead of
writing a body-fat formula of its own.

Its frontmatter says so, under `metadata`, with `audience: app`. That line is
what `aru skills:sync` reads to copy the skill into an application whose
`go.mod` requires this package, and no other skill here carries it: an
application that received the release procedure would follow it.

## Why these exist

The common failure modes when changing this package are a coefficient tuned
until a number looks right, a threshold moved without a line in the changelog, a
call to the clock inside a function that should take `now`, a dependency added
for convenience, and wording that turns an estimate into a diagnosis. None
belongs here.

## Adding your own

A skill in this directory is yours and travels with the repository. Keep it a
procedure rather than a description: a file that says "read the documentation"
never changes what anybody does.
