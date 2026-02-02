<!-- Sync Impact Report:
- Version change: N/A to 1.0.0
- Modified principles:
  - PRINCIPLE_1_NAME to I. Idiomatic Go
  - PRINCIPLE_2_NAME to II. Stdlib-First Dependencies
  - PRINCIPLE_3_NAME to III. Slim APIs
  - PRINCIPLE_4_NAME to IV. Pre-Commit Linting Gates
  - PRINCIPLE_5_NAME to V. Unit Tests Required
- Added sections: None
- Removed sections: None
- Templates requiring updates:
  - UPDATED .specify/templates/plan-template.md
  - UPDATED .specify/templates/spec-template.md
  - UPDATED .specify/templates/tasks-template.md
- Follow-up TODOs:
  - TODO(RATIFICATION_DATE): Original adoption date unknown.
-->
# Chirp Constitution

## Core Principles

### I. Idiomatic Go
Write Go that follows the standard library's conventions: clear naming, explicit
error handling, and straightforward control flow. gofmt output is the baseline;
avoid cleverness that harms readability or toolability.

### II. Stdlib-First Dependencies
Prefer the Go standard library. Add third-party dependencies only when the
stdlib cannot reasonably solve the problem; document the rationale, ownership,
and maintenance risk before adding.

### III. Slim APIs
Keep exported APIs minimal, cohesive, and purpose-built. Introduce interfaces
only when multiple implementations are required; favor concrete types and
package-local helpers to reduce surface area and churn.

### IV. Pre-Commit Linting Gates
All changes MUST pass the repository's pre-commit hooks. At minimum this
includes gofmt and go vet, plus any repo-defined linters. Linting failures
block commits and must be resolved before review.

### V. Unit Tests Required
All new or changed behavior MUST include unit tests using Go's testing
package. Bug fixes require a regression test. Integration tests are optional
but do not replace required unit coverage.

## Project Constraints

- **Language**: Go 1.25.6 (see go.mod).
- **Dependencies**: Stdlib-first; every new non-stdlib dependency requires a
  written justification and maintainer.
- **API Surface**: New exported identifiers and packages require explicit review
  for minimalism and stability.
- **Tooling**: gofmt, go vet, and configured pre-commit hooks are required.

## Development Workflow

- Run pre-commit hooks before committing; resolve all linting errors.
- Run unit tests relevant to the change; prefer go test ./... for broad
  coverage when feasible.
- Keep changes small and reviewable; update docs or comments when behavior
  changes.
- Exceptions to any principle require a written rationale in the plan or PR.

## Governance

- This constitution supersedes all other development practices and templates.
- Amendments require a documented rationale, an updated Sync Impact Report, and
  a semantic version bump (MAJOR: breaking principle changes; MINOR: new or
  expanded principles; PATCH: clarifications only).
- Every spec, plan, and task list MUST include a constitution compliance check
  before implementation.
- Compliance review is mandatory in code review; violations must be justified
  or corrected.
- Runtime guidance lives in README.md and the .specify/templates/ suite.

**Version**: 1.0.0 | **Ratified**: TODO(RATIFICATION_DATE): Original adoption date unknown. | **Last Amended**: 2026-02-02
