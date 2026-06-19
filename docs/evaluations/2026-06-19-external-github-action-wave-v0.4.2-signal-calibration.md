# Released GitHub Action Signal Calibration Wave: v0.4.2 Deeper Sample

Tracking issue: [#327](https://github.com/cervantesh/cervo-mutants/issues/327)

Date: 2026-06-19

This document records a deeper follow-up hosted wave on the released
`github-action@v0.4.2` surface. The earlier mixed-profile released sample in
[#288](https://github.com/cervantesh/cervo-mutants/issues/288) had already
shown that the public GitHub Action path works across the current calibrated
control set. The question here was narrower:

1. keep the same released install surface under test
2. keep the same calibrated mixed-profile target set
3. increase the mutant depth from `10` to `15` per repo while staying bounded
   and CI-shaped
4. measure whether deeper hosted calibration yields more effective review
   signal without reintroducing denominator warnings or runner instability

Committed aggregate artifact:

- [2026-06-19-external-github-action-wave-v0.4.2-signal-calibration-summary.json](2026-06-19-external-github-action-wave-v0.4.2-signal-calibration-summary.json)

## Inputs

Workflow and manifest under test:

- [.github/workflows/external-action-wave.yml](../../.github/workflows/external-action-wave.yml)
- [external-github-action-wave-v0.4.2-signal-calibration.json](external-github-action-wave-v0.4.2-signal-calibration.json)

Verification run:

- successful run: `27853566631`
- successful run URL:
  `https://github.com/cervantesh/cervo-mutants/actions/runs/27853566631`
- branch carrying the manifest:
  `codex/327-next-hosted-wave`
- released surface under test: `github-action@v0.4.2`

## Candidate Mix

This wave intentionally reused the already-calibrated mixed-profile target set
from the earlier released sample. The point was not to widen the repo mix
first. The point was to deepen the hosted signal on a known-good control set.

| Repo | Target | Profile | Special note |
| --- | --- | --- | --- |
| `spf13/pflag` | `./...` | `small-library-control-deeper` | healthy control |
| `tidwall/gjson` | `./...` | `validation-library-control-deeper` | healthy control |
| `prometheus/prometheus` | `./model/labels` | `medium-service-scoped-deeper` | `coverage_prefilter=true` |
| `kubernetes/apimachinery` | `./pkg/api/resource` | `large-multipackage-scoped-retargeted-deeper` | `coverage_prefilter=true`, resolved Go `1.26.0` |

Every job used:

- policy: `ci-balanced`
- budget: `8m`
- sample: `deterministic`
- workers: `2`
- max mutants: `15`

## Result

Aggregate result from run `27853566631`:

- selected repos: `4`
- reports captured: `4`
- missing reports: `0`
- generated mutants: `60`
- covered mutants: `53`
- executed mutants: `53`
- effective mutants: `53`
- killed: `43`
- survived: `10`
- not covered: `7`
- timed out: `0`
- compile errors: `0`
- actionable review units: `9`
- semantic group review units: `2`
- semantic groups formed: `2`
- recommendation entries: `10`
- recommendation review units: `9`
- collapsed recommendation duplicates: `1`
- ledger entries: `7`
- governance suggestions: `15`
- repos with denominator warnings: `0`
- repos with reported failures: `0`

Per-repo result:

| Repo | Profile | Generated | Effective | Killed | Survived | Not covered | Actionable review units | Denominator health |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `pflag-root` | `small-library-control-deeper` | `15` | `15` | `12` | `3` | `0` | `3` | `healthy` |
| `gjson-root` | `validation-library-control-deeper` | `15` | `15` | `12` | `3` | `0` | `2` | `healthy` |
| `prometheus-labels` | `medium-service-scoped-deeper` | `15` | `8` | `8` | `0` | `7` | `0` | `healthy` |
| `apimachinery-resource` | `large-multipackage-scoped-retargeted-deeper` | `15` | `15` | `11` | `4` | `0` | `4` | `healthy` |

## Comparison To The Earlier v0.4.2 Profile Sample

Compared with the earlier released mixed-profile sample from run `27848829038`:

| Metric | Earlier sample | Deeper sample | Change |
| --- | ---: | ---: | ---: |
| Generated mutants | `40` | `60` | `+20` |
| Effective mutants | `37` | `53` | `+16` |
| Killed | `28` | `43` | `+15` |
| Survived | `9` | `10` | `+1` |
| Not covered | `3` | `7` | `+4` |
| Actionable review units | `8` | `9` | `+1` |
| Recommendation entries | `9` | `10` | `+1` |
| Ledger entries | `6` | `7` | `+1` |
| Denominator-warning repos | `0` | `0` | `0` |

That comparison matters. Increasing the mutant depth did increase effective work
substantially, but it did **not** create a proportional increase in new review
burden. Most of the additional depth turned into more killed mutants and more
not-covered rows, not a large new class of actionable survivors.

## Interpretation

### Hosted stability held at the deeper bound

The deeper wave stayed operationally clean:

- no missing reports
- no reported failures
- no timeout or compile-error rows
- no denominator warnings

That is useful because it shows the current released GitHub Action path can
handle a slightly deeper bounded mixed-profile calibration pass without falling
back into the earlier hosted-path fragility.

### Small and validation controls remain the main review-bearing signal source

The smaller control repos stayed healthy and still produced real review work:

- `pflag-root` stayed at `3` actionable review units even after moving from
  `10` to `15` mutants
- `gjson-root` still collapsed repeated boundary-style survivors into `2`
  actionable review units, with `1` semantic group and `1` collapsed duplicate

That is stronger evidence that the semantic grouping and recommendation surfaces
remain stable under a slightly deeper hosted sample, not just under the
earlier smaller control wave.

### `apimachinery-resource` scales cleanly and remains review-bearing

The large scoped target remained the healthiest broader-profile review lane in
this wave:

- `generated=15`
- `effective=15`
- `survived=4`
- `actionable_review_units=4`
- resolved Go version remained `1.26.0`

That means the broader-profile hosted path is no longer just "barely working"
on this target. It continues to produce bounded review signal as depth rises.

### `prometheus-labels` is now the main calibration limiter

`prometheus-labels` stayed operationally healthy, but its extra depth mostly
became additional not-covered rows:

- earlier sample: `effective=7`, `not_covered=3`, `survived=0`
- deeper sample: `effective=8`, `not_covered=7`, `survived=0`

That is not a runner regression. It is a target-shape and coverage-yield
question. The next calibration decision for the medium-service profile should
therefore focus on target selection or coverage reach, not on hosted-path
stability or semantic ranking weights.

## What This Proves And What It Does Not

This wave is enough to support these narrower claims:

1. the released `github-action@v0.4.2` path stays stable on the current
   calibrated mixed-profile control set at `15` mutants per repo
2. deeper hosted calibration can raise effective-mutant yield materially
   without reintroducing denominator warnings
3. the current recommendation and semantic-grouping surfaces still behave
   coherently under that deeper bound

This wave does **not** prove:

1. that `15` mutants is the right hosted default for every public adoption lane
2. that broader service-repo targets now yield rich survivor diversity
3. that semantic-triage calibration is complete across more than this curated
   control set

## Conclusion

Issue `#327` now has initial hosted evidence for the next maturity step:

- the repo can gather deeper external calibration evidence on the released
  public GitHub Action surface
- the strongest mixed-profile review signal still comes from the smaller
  controls plus the retargeted `apimachinery` lane
- the main remaining calibration gap has narrowed to medium-service target
  yield, not hosted execution reliability

The next justified follow-up is not heuristic retuning first. It is deciding
whether the medium-service lane should be retargeted or coverage-shaped so that
future hosted adoption waves gain more review-bearing signal instead of mostly
more `not_covered` rows.
