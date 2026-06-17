# Small Library Example

This example models the lowest-friction adoption path for a small Go package or
shared utility library.

## When to copy this

Use this template when:

- the repo is small enough that `./...` is a realistic mutation target
- you want a single PR workflow first
- you need baseline-first adoption instead of a hard score gate on day one

## Suggested rollout

1. Run a dry discovery locally:

   ```powershell
   cervomut run ./... --dry-run --out .cervomut/preview
   ```

2. Establish the first accepted baseline:

   ```powershell
   cervomut run ./... --policy ci-fast --budget 5m --out .cervomut/pr
   cervomut baseline update
   ```

3. Use the first-party workflow in `.github/workflows/cervomut.yml` for PR
   feedback.

## Why this shape

- `ci-fast` keeps the operator set narrow and reviewable.
- reports stay compact: summary, JSON, JUnit, and GitHub summary.
- the config leaves `ci.fail_under=0` because the primary gate is baseline
  regression, not raw score vanity.
