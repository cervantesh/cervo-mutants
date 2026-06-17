# Repository Instructions

All work in this repository must be tracked through a GitHub issue before
implementation starts.

- Use an existing issue when one already covers the scope.
- Create a new issue before making changes when no issue exists.
- Keep the issue updated with scope, decisions, tests, PR links, actual time,
  and deviation when applicable.
- Reference the issue from branch names, commits, and pull requests when
  practical.

## Mutation Tool Comparisons Must Be Apples-To-Apples

When running CervoMutants against Gremlins, gomu, go-mutesting, or any other
mutation-testing tool, every result must record target semantics explicitly.

- Store the manifest target, effective target, and target mode for every tool.
- Do not compare `cervomut run ./...` with `gremlins unleash .` as if they were
  equivalent.
- For apples-to-apples package-root studies, normalize all tools to the same
  effective target, for example `.` when the manifest target is `./...`.
- Mark runs as diagnostic, not comparative, when effective targets or target
  modes differ.
- Preserve denominator health fields: killed, survived, not covered, timed out,
  errors, test efficacy, mutation coverage, and score denominator.
- If a final CervoMutants report is missing, inspect
  `partial-mutation-report.json` before reporting `no_report`.

The reusable protocol is documented in
`docs/evaluations/tool-comparison-protocol.md`, and the runnable harness is
described in `docs/evaluations/comparison-harness.md`. Agents should follow
both before launching or summarizing comparison runs.

