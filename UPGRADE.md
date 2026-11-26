# Upgrade Guide

## v0.1.0

First release: nothing to change. An application that estimated body
composition by hand replaces its own formulas with `bodycomp.Estimate`, keeps
the `Result` and the `Session` (both serialise to JSON) in its own storage and
under its own policies, and passes `now` in wherever a function needs a date.

