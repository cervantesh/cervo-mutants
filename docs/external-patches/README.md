# External Mutation Tool Windows Patches

These patches were produced during issue #10 while comparing CervoMutants with
other Go mutation-testing tools on Windows/OneDrive.

They are not vendored dependencies of CervoMutants. They document the minimal
local changes needed to make the exact studied versions run in the same Windows
environment where the upstream binaries failed.

## Tools

- `gomu v0.2.0`
  - Fixes temp mutant directory names derived from absolute Windows paths such
    as `C:\...`.
  - Validation: patched binary completed against `github.com/spf13/cobra/doc`.

- `go-mutesting v2.6.13`
  - Fixes mutation temp paths derived from absolute Windows paths.
  - Removes the runtime dependency on an external Unix `diff` executable by
    using an internal fallback diff.
  - Validation: patched binary completed against `github.com/spf13/cobra/doc`.

## Apply

Apply from each upstream repository root:

```text
git apply path/to/gomu-v0.2.0-windows-paths.patch
git apply path/to/go-mutesting-v2.6.13-windows-paths.patch
```


