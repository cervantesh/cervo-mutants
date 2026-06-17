# Large Repo CI Example

This example is a reference workspace for teams that need bounded mutation
execution before they can trust broad `./...` runs in CI.

## When to copy this

Use this template when:

- the repository is large enough that a single mutation lane is too noisy or slow
- you need deterministic shards or file caps
- PR and nightly lanes must use different targets or density limits

## Suggested rollout

Pull request lane:

```powershell
cervomut run ./... --policy ci-fast --slice-by package --shard 1/4 --max-files-per-run 20 --sample deterministic --out .cervomut/pr
```

Nightly lane:

```powershell
cervomut run ./... --policy nightly --slice-by file --shard 3/12 --max-mutants-per-package 25 --sample deterministic --report summary,json,junit,html,sarif,github-summary --out .cervomut/nightly
```

Campaign lane:

```powershell
cervomut run ./pkg/catalog ./pkg/pricing --policy campaign --workers 2 --out .cervomut/campaign
```

## Why this shape

- sharding stays deterministic and reviewable
- file and package caps keep the denominator meaningful under CI budgets
- the workflow demonstrates matrix slicing without claiming that every large
  repo should mutate the exact same scope
