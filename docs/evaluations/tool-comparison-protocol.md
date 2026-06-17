# Mutation Tool Comparison Protocol

This protocol exists so humans and coding agents do not accidentally compare
different mutation-testing scopes as if they were equivalent.

## Required Classification

Every comparison row must record:

- `target`: the manifest target requested by the study, for example `./...`.
- `effective_target`: the target actually passed to the tool, for example `.`.
- `target_mode`: the normalization strategy, usually `manifest` or
  `package-root`.
- `manifest_equivalent`: whether the effective target is the same scope as the
  manifest target.
- `apples_to_apples_key`: stable grouping key made from target mode and
  effective target.

A tool comparison is apples-to-apples only when the compared rows have the same
`effective_target` and the same `target_mode`.

`manifest_equivalent=false` is allowed, but it changes the interpretation:
the tools can be comparable to each other while not representing the original
manifest scope. This is the correct shape for a package-root Gremlins study
where `./...` is normalized to `.` for every tool.

## Valid Study Modes

### Manifest Mode

Use this when the study asks whether each tool can run the manifest target as
written.

```text
manifest target: ./...
CervoMutants effective target: ./...
Gremlins effective target: ./...
gomu effective target: ./...
go-mutesting effective target: ./...
target_mode: manifest
```

If a tool cannot interpret the manifest target correctly, the result is an
operational finding. Do not silently normalize only that tool.

### Package-Root Mode

Use this when the study asks for a fair package-root comparison, especially
against Gremlins.

```text
manifest target: ./...
CervoMutants effective target: .
Gremlins effective target: .
gomu effective target: .
go-mutesting effective target: .
target_mode: package-root
manifest_equivalent: false
```

This is apples-to-apples between tools, but not equivalent to a full `./...`
module campaign.

### Diagnostic Mode

Use this only to understand tool behavior. Examples:

```text
CervoMutants effective target: ./...
Gremlins effective target: .
```

Diagnostic results must not be summarized as "Tool A is faster/better than Tool
B" without a target-semantics warning.

## Required Metrics

Every result must carry the denominator components:

- generated or total mutants;
- killed;
- survived;
- not covered;
- timed out;
- errors or not viable mutants;
- test efficacy over killed plus survived;
- mutation coverage where available;
- denominator health warnings.

Raw mutation score is not enough. A high score with a tiny effective denominator
and many timed-out or not-covered mutants is an unhealthy result.

## CervoMutants Partial Reports

When `mutation-report.json` is absent because a run timed out or was killed,
agents must inspect `partial-mutation-report.json` before recording `no_report`.

Use the final report when present. Use the partial report only as a fallback and
mark `partial_report_used=true`.

## Harness Defaults

For future CervoMutants-vs-Gremlins studies:

```powershell
.\scripts\compare-tools-pool.ps1 `
  -Tools cervomut,gremlins `
  -CompareTargetMode package-root `
  -GremlinsTargetMode package-root `
  -Workers 2 `
  -TimeoutSeconds 600
```

For all-tool studies, use the same `CompareTargetMode` for CervoMutants, gomu,
and go-mutesting. Gremlins may additionally set `GremlinsTargetMode
package-root` for compatibility, but it must not be the only normalized tool in
a fairness claim.

## Reporting Rule

Every comparison summary must include one of these labels:

- `apples_to_apples=true`: same effective target and target mode.
- `manifest_equivalent=false`: comparable between tools, but not the original
  manifest scope.
- `not_comparable`: effective targets or target modes differ.

When in doubt, mark the row not comparable and explain why.

