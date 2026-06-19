# Compatibility Policy

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/93

This document defines which CervoMutants surfaces are treated as compatibility
contracts and what kind of change is allowed on each one.

## Scope

This policy applies to:

- documented CLI commands and flags
- report outputs intended for CI, automation, or long-lived integrations
- documented library extension seams used to customize generation, suppression,
  or ranking
- the daemon/worker protocol surface

## Compatibility Levels

### Supported

Supported surfaces are expected to remain stable across normal releases.

Changes are allowed when they are:

- additive
- bug fixes that align behavior with documented intent
- deprecations that are announced in release notes and upgrade notes first

Removals or semantic breaks on supported surfaces must not happen silently.

### Additive

Additive means existing valid usage keeps working while new fields, commands,
or options are introduced.

Examples:

- adding optional JSON fields under the existing report schema version
- adding a new CLI command
- adding a new enum value while keeping older values unchanged

### Deprecated

Deprecated surfaces still work for at least one public release after the
deprecation is announced, unless there is a security or correctness reason that
makes immediate removal necessary.

Deprecations must be called out in:

- `CHANGELOG.md`
- the matching `docs/upgrade-notes/vX.Y.Z.md`

### Experimental

Experimental surfaces are opt-in and do not carry backward-compatibility
guarantees yet.

They may change, be renamed, or be removed between releases without following
the supported-surface deprecation window.

## CLI Policy

The following CLI surface is treated as supported when documented in `README.md`
and not marked experimental:

- top-level commands
- subcommands
- documented flags

Rules:

- adding new commands or flags is additive
- removing or renaming a documented non-experimental command or flag requires a
  deprecation notice first
- changing the meaning of an existing documented flag is a breaking change and
  must be called out explicitly in release notes and upgrade notes

Undocumented debug-only behavior is not a compatibility promise.

## Report Schema Policy

The public report schema is currently `schema_version: "1"`.

For schema v1:

- new fields must be additive and optional for older consumers
- existing fields must not change meaning silently
- existing enum values and status names must remain stable
- removing fields, changing meanings, or changing requiredness is a breaking
  change

Breaking report changes require:

- a new schema version
- explicit migration notes in the release changelog and upgrade note

This policy covers the JSON report and other documented report projections that
are described as public compatibility surfaces.

## Extension Library Policy

The documented library seams in [docs/extensibility.md](extensibility.md) are a
supported integration surface when used through their published interfaces and
helpers.

That currently includes:

- `mutator.Generator`
- `mutator.DefaultGenerator()`
- `mutator.ChainGenerators(...)`
- `engine.SuppressionEvaluator`
- `engine.DefaultSuppressionEvaluator(cfg)`
- `engine.ChainSuppressionEvaluators(...)`
- `engine.SurvivorRanker`
- `engine.DefaultSurvivorRanker()`
- `engine.NewWithOptions(cfg, ...)`

Rules:

- additive helpers, fields, or metadata are allowed when existing integrations
  keep working
- semantic changes to documented extension behavior must be called out in
  release notes and upgrade notes
- removals, renames, or signature breaks on these seams require a deprecation
  path unless there is a correctness or security exception

What is not promised yet:

- a standalone triage-plugin protocol
- CLI-level plugin distribution or discovery
- compatibility for undocumented internal helpers outside the published seams

The repository keeps focused tests around these seams so refactors fail loudly
when custom generator, suppression, or ranking scenarios break.

## Daemon And Worker Policy

`cervomut daemon` and `cervomut worker` are currently experimental.

That means:

- the JSON-lines protocol is not a supported compatibility surface yet
- there is no promise of backward compatibility between releases
- protocol versioning must exist before this surface can graduate to supported

Promotion from experimental to supported requires explicit documentation and a
versioned protocol contract.

## Support Matrix Policy

OS and Go-version support claims are governed by
[go-toolchain-compatibility.md](go-toolchain-compatibility.md).

An environment is officially supported only when it is documented there and
covered by the corresponding automated validation workflow.

## Release Process Requirements

Every release that changes a supported surface must document that change in:

- `CHANGELOG.md`
- the matching `docs/upgrade-notes/vX.Y.Z.md`

The release workflow already depends on those files, so missing upgrade
communication blocks publication instead of relying on memory.

Main CI also runs an upgrade smoke from the latest public release archive into
the current source tree so supported report and baseline workflows fail loudly
when a release-to-release handoff breaks.

The release workflow also validates the documented `go install` and
archive-install paths with a real installed binary and exercises report
generation plus report-reading commands before publication.
