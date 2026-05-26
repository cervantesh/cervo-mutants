# cervo-mutant

`cervo-mutant` is a Go mutation testing library and CLI for CervoSoft projects.
It provides the `cervomut` binary plus importable packages for discovery,
mutation, execution, reporting, baseline comparison, and daemon-ready job
contracts.

Tracking issue: https://gitea.cervbox.synology.me/CervoSoft/cervo-mutant/issues/1

## Quick Start

```powershell
go install gitea.cervbox.synology.me/CervoSoft/cervo-mutant/cmd/cervomut@latest
cervomut init
cervomut doctor
cervomut run ./... --budget 10m --sample deterministic
```

## Commands

- `cervomut init`
- `cervomut doctor`
- `cervomut affected ./...`
- `cervomut run ./...`
- `cervomut run ./... --dry-run`
- `cervomut baseline update`
- `cervomut baseline compare`
- `cervomut report summary`
- `cervomut report survivors`
- `cervomut report open`
- `cervomut show <mutant-id>`
- `cervomut explain <mutant-id> --format text|json`
- `cervomut list-mutators`
- `cervomut daemon`
- `cervomut worker`

## Defaults

The default policy is baseline-first adoption:

- `ci.fail_under` is `0`.
- baseline regression and new survivors are tracked.
- quarantine entries require owner, issue, reason, and expiry.
- reports include summary, JSON schema v1, JUnit XML, and HTML.
- execution uses temp workdirs so source workspaces are not mutated.

## Development

```powershell
go test ./...
go run ./cmd/cervomut list-mutators
```

