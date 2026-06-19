# Released-Surface Triage Yield From v0.4.0 Adoption Artifacts

Tracking issue: [#237](https://github.com/cervantesh/cervo-mutants/issues/237)

Date: 2026-06-19

This note measures recommendation and semantic-triage yield from the first
released-surface GitHub Action adoption artifacts.

The goal is narrower than "semantic triage is fully calibrated." The goal is
to answer three practical questions from real released artifacts:

1. did the released wave produce reviewable triage output at all
2. which weak results came from low-signal inputs versus heuristic behavior
3. which follow-ups are justified by the released evidence itself

## Evidence Base

Committed evidence used here:

- [2026-06-19-external-github-action-wave-v0.4.0.md](2026-06-19-external-github-action-wave-v0.4.0.md)
- [2026-06-19-external-github-action-wave-v0.4.0-summary.json](2026-06-19-external-github-action-wave-v0.4.0-summary.json)

Direct artifact inspection from successful hosted run `27836288517`:

- `external-wave-pflag-root`
  - `test-recommendations.md`
  - `semantic-triage-ledger.json`
  - `governance-review.md`
- `external-wave-gjson-root`
  - `test-recommendations.md`
  - `semantic-triage-ledger.json`
  - `governance-review.md`
- `external-wave-logrus-root`
  - `test-recommendations.md`
  - `semantic-triage-ledger.json`
  - `governance-review.md`

## Aggregate Released Yield

The released hosted wave summary preserved these aggregate triage fields:

| Metric | Value |
| --- | ---: |
| Repositories selected | `3` |
| Reports captured | `3` |
| Survived mutants | `6` |
| Actionable review units | `5` |
| Equivalent-risk survivors | `4` |
| Semantic-group review units | `2` |
| Collapsed semantic duplicates | `1` |
| Semantic groups formed | `2` |
| Recommendation entries | `6` |
| Ledger entries | `3` |
| Governance suggestions | `8` |
| Repos with denominator warnings | `1` |

That is materially different from the earlier hosted-wave evidence that
collapsed to zero actionable review units. The released-surface wave now
produces enough positive signal to evaluate triage behavior instead of only
workflow plumbing.

## Per-Repo Interpretation

| Repo | Signal class | Survivors | Review units | Recommendations | Ledger | Governance | Interpretation |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `pflag-root` | healthy signal | `3` | `3` | `3` | `1` | `2` | recommendation and triage outputs align closely; one high-equivalent-risk survivor is flagged without duplicate collapse |
| `gjson-root` | healthy signal with semantic collapse | `3` | `2` | `3` raw | `2` | `4` | semantic grouping collapses one duplicate review path; recommendations stay local and coherent |
| `logrus-root` | low-signal input | `0` | `0` | `0` | `0` | `2` | denominator warnings and `not_covered` rows dominate; this is input friction, not a triage-ranking failure |

## Artifact-Level Observations

### `pflag-root`

The released artifacts show a clean "real work" shape:

- `3` surviving mutants
- `3` actionable review units
- `3` recommendation entries
- `1` ledger entry

All three recommendations pointed to nearby local tests and used the same
strategy:

- `tighten-branch-assertions`

The ledger only flagged one survivor as high equivalence-risk:

- `flag.go:1002:conditionals-boundary:ecc1235bc48c`

That means the triage layer is not collapsing work indiscriminately here. It
is flagging one review-once family while still leaving the other survivors as
independent work.

### `gjson-root`

This repo is the clearest proof that semantic grouping is doing real work.

The released artifacts show:

- `3` surviving mutants
- `2` true actionable survivors
- `1` collapsed semantic duplicate
- `1` semantic group with size `2`
- `3` raw recommendation entries

The ledger grouped two survivors under the same `len boundary` family:

- `gjson.go:1003:conditionals-boundary:387ffa183f38`
- `gjson.go:1009:conditionals-boundary:f79ef8efa4b8`

The recommendations were still emitted per survivor, but the semantic-group
evidence says one good review can collapse two similar rows.

This is a positive functional signal:

- semantic grouping is not just theoretical in committed released artifacts
- the recommendations remain plausible and local to `gjson_test.go`

It also exposes a measurement gap:

- the wave summary says `recommendation_entries=3`
- the real human review workload is closer to `2` recommendation-bearing review
  units after collapse

That is the basis for follow-up issue [#242](https://github.com/cervantesh/cervo-mutants/issues/242).

### `logrus-root`

`logrus-root` is the main low-signal case in this released wave.

The summary shows:

- `survived=0`
- `not_covered=6`
- denominator warnings:
  - `not_covered_exceeds_effective`
  - `high_score_poor_denominator_health`

The important point is what the triage surfaces did **not** do:

- no actionable survivors
- no recommendation entries
- empty semantic-triage ledger

This is exactly the behavior we want from a low-signal input. The released
artifact does not show noisy fabricated review advice. It shows a repo where
coverage and denominator health are the bottleneck.

The same artifact still emitted `2` governance suggestions, but those came from
`not_covered` high-equivalent-risk rows in `ci/magefile.go`. That is audit
signal, not survivor workload.

## Findings

### 1. The released artifacts finally produce enough positive triage signal to evaluate behavior, not just absence of behavior

The released wave is no longer a zero-yield hosted pass. It produced:

- `5` actionable review units
- `2` semantic groups
- `6` recommendation entries
- `3` ledger entries

That is enough to support a real calibration note grounded in what adopters
would actually consume from the released GitHub Action path.

### 2. The remaining low-signal problem is localized to one repo, not the whole released wave

`logrus-root` is denominator-poor. `pflag-root` and `gjson-root` are not.

That matters because it narrows the interpretation:

- this is not a released-surface failure
- this is not evidence that semantic triage collapses to no-op on public repos
- this is one remaining candidate-quality or target-shape problem inside an
  otherwise reviewable released wave

### 3. Recommendation quality looks coherent on this released sample

The positive-signal repos produced recommendation behavior that is easy to
defend:

- all recommendations point to nearby local tests
- all suggested strategies are branch/boundary assertion tightening
- the suggestions line up with the actual operator families seen in the wave

There is no released evidence here of obviously noisy or off-target
recommendation behavior.

### 4. Semantic grouping is working, but the current summary can overstate recommendation workload

The clearest example is `gjson-root`:

- `3` surviving mutants carried recommendations
- semantic grouping collapsed that to `2` real review units

The current summary persists raw recommendation count correctly, but not the
review-unit count after semantic collapse.

That is not a ranking bug. It is a report-contract gap now tracked in
[#242](https://github.com/cervantesh/cervo-mutants/issues/242).

### 5. Governance suggestion totals are useful audit signal, but they are not the same thing as actionable review workload

The released wave produced:

- `8` governance suggestions
- `5` actionable review units

The mismatch is not accidental noise. It comes from the fact that governance
templates can be generated for non-survivor states as audit hints. `logrus-root`
proves that with `2` governance suggestions and `0` survivors.

That is a summary-interpretation gap, not a triage failure. It now has a
follow-up in [#243](https://github.com/cervantesh/cervo-mutants/issues/243).

### 6. The released sample is still too narrow to overclaim broad heuristic calibration

This wave is still dominated by boundary and negation style survivors. The
released evidence does **not** justify claiming broad calibration across:

- platform-sensitive survivors
- non-progress loop mutants
- larger semantic-group families
- more diverse operator mixes

The correct conclusion is narrower: the recommendation and grouping layers look
credible on this released branch/boundary-heavy sample.

## Low-Signal Inputs Versus Heuristic Defects

The released evidence separates these two concerns more clearly now:

### Low-signal input evidence

- `logrus-root` denominator warnings
- `6` `not_covered` rows
- zero survivors, recommendations, and ledger entries there

This is an input-yield problem.

### Heuristic-quality evidence

- `pflag-root` shows actionable recommendations and one bounded equivalence-risk
  flag without collapse noise
- `gjson-root` shows semantic collapse working and preserving local test
  suggestions
- no repo shows fabricated review advice when survivors are absent

This is positive evidence for the current heuristic behavior on the released
sample.

## Follow-Up Issues Opened From This Evidence

- [#242](https://github.com/cervantesh/cervo-mutants/issues/242) Persist
  review-unit recommendation counts in adoption-wave summaries
- [#243](https://github.com/cervantesh/cervo-mutants/issues/243) Split
  governance suggestion counts by mutant status in adoption-wave summaries

These are both narrow follow-ups justified directly by the released artifacts.
Neither issue assumes the triage engine is wrong. Both address how released
evidence is summarized for humans.

## Conclusion

The released `v0.4.0` adoption artifacts support a stronger claim than the
earlier hosted-wave evidence:

- semantic triage now produces real reviewable yield on released public-wave
  artifacts
- the remaining low-signal case is isolated, not dominant
- the next justified follow-ups are summary-shape refinements, not panic-tuning
  of recommendation heuristics

That is the right outcome for `#237`: use released evidence to narrow the next
calibration step, not to relitigate whether the released GitHub Action path
works at all.
