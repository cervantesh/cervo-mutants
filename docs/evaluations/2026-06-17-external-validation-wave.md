# External Validation Wave

Tracking issue: #95

Date: 2026-06-17

This document records the first reproducible external validation wave for
CervoMutants against public Go repositories outside this repository.

The goal was not to claim production adoption by upstream maintainers. The goal
was to prove that the current public release can be applied, without custom
patches, to a small but real set of public repositories and produce bounded
mutation results with auditable artifacts.

## Wave Shape

The wave ran through:

- [external-validation-wave-5-campaign.json](external-validation-wave-5-campaign.json)
- `cervomut pool campaign`

The exact repositories were:

| Repo | Domain | Target |
| --- | --- | --- |
| `cobra` | CLI | `./doc` |
| `pflag` | CLI | `./...` |
| `logrus` | logging | `./...` |
| `uuid` | utility | `./...` |
| `gjson` | parser | `./...` |

Bounded execution settings:

- `run_mutation: true`
- `max_mutants: 10`
- `workers: 2`
- one Windows host

## Command

```powershell
go build -o .cervomut/tools/cervomut-wave.exe ./cmd/cervomut
.cervomut/tools/cervomut-wave.exe pool campaign `
  --file docs/evaluations/external-validation-wave-5-campaign.json `
  --resume `
  --cervomutants .cervomut/tools/cervomut-wave.exe
```

## Result

Artifacts produced:

- [2026-06-17-external-validation-wave-campaign-summary.json](2026-06-17-external-validation-wave-campaign-summary.json)
- [2026-06-17-external-validation-wave-summary.json](2026-06-17-external-validation-wave-summary.json)
- per-repo clone, baseline, dry-run, and mutation logs under
  `.cervomut/external-validation-wave/work/public-wave-smoke/` in the local
  workspace used for the wave

Aggregate result:

- repos selected: `5`
- baseline test success: `5/5`
- dry-run success: `5/5`
- bounded mutation success: `5/5`
- generated mutants executed in wave: `50`
- killed: `37`
- survived: `13`
- not covered: `0`
- total elapsed time: `107.257s`

Per-repo result:

| Repo | Baseline s | Dry-run s | Mutation s | Elapsed s | Score |
| --- | ---: | ---: | ---: | ---: | ---: |
| `cobra` | `4.649` | `0.034` | `8.465` | `13.958` | `40` |
| `pflag` | `2.509` | `0.134` | `10.342` | `13.760` | `60` |
| `logrus` | `12.590` | `0.065` | `9.476` | `22.733` | `80` |
| `uuid` | `2.509` | `0.046` | `8.699` | `11.811` | `100` |
| `gjson` | `11.385` | `0.468` | `32.498` | `44.995` | `90` |

## Findings

1. The current public tool is able to clone, baseline-test, dry-run, and
   execute bounded mutation successfully across all five selected public repos
   on Windows without repo-specific patches.
2. The wave did not degrade into denominator noise. There were no `not_covered`
   outcomes in this bounded sample and no timeout-driven partial results.
3. The repos were not trivial duplicates of one domain. The wave covered CLI,
   logging, parser, and utility code, and still completed cleanly.
4. Runtime variance is real but manageable. `gjson` was materially slower than
   the others, which is useful operational evidence for future public guidance.
5. The survivor range stayed meaningful. The wave produced scores from `40` to
   `100`, which means the run still surfaced reviewable variation instead of
   only yielding saturated pass/fail behavior.

## Threats To Validity

1. This was a public-repo validation wave, not direct maintainer feedback from
   those upstream projects.
2. The wave was bounded to `10` mutants per repo, so it validates first-run
   adoption viability more than large-depth mutation campaigns.
3. The selected repos were small to medium libraries, not large operational Go
   services or monorepos.
4. The run happened on one Windows host, so it does not replace the official
   CI support matrix.

## Conclusion

This wave is enough to claim a first external public-repo validation pass:

- the tool works outside its own repository
- the workflow is reproducible from committed artifacts
- the findings are concrete enough to inform public trust claims

It does not yet prove broad maintainer adoption, but it closes the narrower
gap between "strong local beta" and "publicly unvalidated tool."
