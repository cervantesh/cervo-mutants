# Medium Service Example

This example models a service-sized repository with several internal packages
and a need for richer review outputs than the smallest-library path.

## When to copy this

Use this template when:

- the repo has multiple packages or bounded internal domains
- you want survivor recommendations and HTML review output
- you can afford `ci-balanced` in pull requests and a broader nightly run

## Suggested rollout

Pull request lane:

```powershell
cervomut run ./... --policy ci-balanced --budget 10m --coverage-prefilter --report summary,json,junit,github-summary --out .cervomut/pr
```

Nightly lane:

```powershell
cervomut run ./... --policy nightly --budget 20m --report summary,json,junit,html,sarif,github-summary --out .cervomut/nightly
```

## Why this shape

- coverage-aware selection cuts irrelevant tests earlier
- HTML and recommendations matter once survivors stop being trivial
- the workflow shows how to keep PR and nightly lanes distinct with the
  first-party GitHub Action instead of custom install steps per job
