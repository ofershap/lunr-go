# Contributing

Issues and pull requests are welcome.

## Setup

Go 1.26 or newer. Standard library only; no other dependencies for the
library or tests. Regenerating the differential corpus needs Node (pinned
lunr@2.3.9 in testdata/gen/, committed lockfile) and is not part of the
normal test run.

## Checks

Run everything CI runs before opening a PR:

```bash
go vet ./...
gofmt -l .
go test -race ./...
```

## Rules

- Behaviour must match lunr.js 2.3.9 bit-exactly: scores compare with exact
  float64 equality and index serialization is byte-identical on the tested
  corpus. If upstream changes, extend the corpus and regenerate fixtures in
  the same change; intentional divergences (currently only the fuzzy
  edit-distance bounds) get a pinned regression test and a README note.
- Keep the dependency tree empty.
- Claims in the README must match what the tests demonstrate.
