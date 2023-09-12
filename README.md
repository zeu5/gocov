# gocov

Library to read go coverage data programmatically

## Why?

Because I want coverage information at runtime. go API's are closed and the tools repo has not been updated with the new coverage data structures.

## How?

Credit to the go team, I reused most of their code (internal/coverage) and defined wrapper data structures.

## Problems?

Tightly coupled with the encoding format of the Coverage information. i.e. The go compiler version.

## TODO

- [ ] Need to add a version check
- [ ] Merge and diff `CoverageData`
