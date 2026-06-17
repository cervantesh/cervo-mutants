# Sonar Analysis

Tracking issue: https://github.com/cervantesh/cervo-mutants/issues/31

CervoMutant includes `sonar-project.properties` so SonarQube or SonarCloud can
ingest Go sources, tests, and coverage.

## Local Preparation

Generate coverage before running the scanner:

```powershell
go test ./... -coverprofile=coverage.out
```

Then run SonarScanner from the repository root:

```powershell
sonar-scanner
```

The scanner is not vendored in this repository. Install it on the CI runner or
developer machine and provide server credentials through environment variables
or the runner's secret store.

Common environment variables:

```powershell
$env:SONAR_HOST_URL = "https://sonar.example.com"
$env:SONAR_TOKEN = "<token>"
sonar-scanner
```

## Scope

The Sonar project analyzes:

- `cmd`
- `pkg`

It excludes:

- vendored dependencies;
- `.cervomut` outputs;
- fixtures and testdata;
- generated report artifacts such as JSON, XML, and HTML.

## Local Quality Pass

When SonarScanner is unavailable locally, run the equivalent local checks:

```powershell
gofmt -w cmd pkg
go test ./... -coverprofile=coverage.out
go vet ./...
staticcheck ./...
```

`staticcheck` can be installed with:

```powershell
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Do not treat local checks as a full Sonar replacement. They are the fast local
gate before publishing scanner results.

## 2026-05-30 Local Sonar Pass

Server:

```text
http://localhost:9000
SonarQube Community Build 26.5.0.122743
SonarScanner CLI 8.0.1.6346
```

Initial result before the issue #31 quality work:

| Metric | Value |
| --- | ---: |
| Quality Gate | OK |
| Bugs | 0 |
| Vulnerabilities | 0 |
| Coverage | 69.2% |
| Duplicated lines density | 0.9% |
| Open code smells | 48 |

Implemented in this pass:

- Added Sonar project configuration.
- Added local Sonar instructions.
- Added tests for `baseline`, `cache`, `runner`, and `selecttest`.
- Fixed `runner.GoTestRunner` compile-error classification.
- Fixed malformed daemon worker fixture test.
- Made `baseline.Save` create parent directories.
- Documented intentional no-op cleanup paths for process limits and mutation
  preparation.

Final verified result after the issue #31 quality work:

| Metric | Value |
| --- | ---: |
| Bugs | 0 |
| Vulnerabilities | 0 |
| Coverage | 90.7% |
| Duplicated lines density | 0.0% |
| Open code smells | 0 |

The final pass also ran:

```powershell
go test ./... -coverprofile=coverage.out
go vet ./...
staticcheck ./...
```
