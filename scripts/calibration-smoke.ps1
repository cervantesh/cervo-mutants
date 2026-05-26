param(
    [string]$Manifest = "docs/evaluations/go-repo-pool-40.json",
    [string]$WorkRoot = "$env:TEMP/cervomut-go-pool-40",
    [string[]]$Names = @(),
    [int]$Limit = 0,
    [switch]$RunMutation,
    [int]$MaxMutants = 10,
    [int]$Workers = 2,
    [int]$CloneTimeoutSeconds = 180,
    [int]$TestTimeoutSeconds = 120,
    [int]$DryRunTimeoutSeconds = 120,
    [int]$MutationTimeoutSeconds = 300
)

$ErrorActionPreference = "Stop"

$manifestPath = Resolve-Path -LiteralPath $Manifest
$manifestData = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
$repos = @($manifestData.repos)
if ($Names.Count -gt 0) {
    $wanted = @{}
    foreach ($name in $Names) {
        $wanted[$name] = $true
    }
    $repos = @($repos | Where-Object { $wanted.ContainsKey($_.name) })
}
if ($Limit -gt 0) {
    $repos = @($repos | Select-Object -First $Limit)
}

New-Item -ItemType Directory -Path $WorkRoot -Force | Out-Null
$results = @()

function Invoke-LoggedCommand {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$WorkingDirectory,
        [string]$LogPath,
        [int]$TimeoutSeconds
    )
    $stdout = "$LogPath.stdout"
    $stderr = "$LogPath.stderr"
    Remove-Item -LiteralPath $stdout, $stderr, $LogPath -Force -ErrorAction SilentlyContinue
    $proc = Start-Process -FilePath $FilePath -ArgumentList $Arguments -WorkingDirectory $WorkingDirectory -NoNewWindow -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    if (!$proc.WaitForExit($TimeoutSeconds * 1000)) {
        try { $proc.Kill($true) } catch { $proc.Kill() }
        [void]$proc.WaitForExit()
        "timed out after ${TimeoutSeconds}s" | Set-Content -LiteralPath $LogPath
        return 124
    }
    $parts = @()
    if (Test-Path -LiteralPath $stdout) { $parts += Get-Content -LiteralPath $stdout -Raw }
    if (Test-Path -LiteralPath $stderr) { $parts += Get-Content -LiteralPath $stderr -Raw }
    ($parts -join [Environment]::NewLine) | Set-Content -LiteralPath $LogPath
    Remove-Item -LiteralPath $stdout, $stderr -Force -ErrorAction SilentlyContinue
    return $proc.ExitCode
}

foreach ($repo in $repos) {
    $repoDir = Join-Path $WorkRoot $repo.name
    $started = Get-Date
    $result = [ordered]@{
        name = $repo.name
        url = $repo.url
        target = $repo.target
        lane = $repo.lane
        domain = $repo.domain
        clone = "pending"
        baseline_exit = $null
        baseline_seconds = $null
        dry_run_exit = $null
        dry_run_seconds = $null
        mutation_exit = $null
        mutation_seconds = $null
        mutants = $null
        killed = $null
        survived = $null
        not_covered = $null
        score = $null
        notes = ""
    }

    try {
        if (!(Test-Path -LiteralPath $repoDir)) {
            $cloneExit = Invoke-LoggedCommand -FilePath "git" -Arguments @("clone", "--depth", "1", $repo.url, $repoDir) -WorkingDirectory $WorkRoot -LogPath (Join-Path $WorkRoot "$($repo.name)-clone.log") -TimeoutSeconds $CloneTimeoutSeconds
            if ($cloneExit -ne 0) {
                $result.clone = "failed"
                $result.notes = "clone exit $cloneExit"
                $results += [pscustomobject]$result
                continue
            }
        }
        $result.clone = "ok"
        Push-Location $repoDir
        try {
            $sw = [Diagnostics.Stopwatch]::StartNew()
            $result.baseline_exit = Invoke-LoggedCommand -FilePath "go" -Arguments @("test", $repo.target) -WorkingDirectory $repoDir -LogPath (Join-Path $WorkRoot "$($repo.name)-baseline.log") -TimeoutSeconds $TestTimeoutSeconds
            $sw.Stop()
            $result.baseline_seconds = [math]::Round($sw.Elapsed.TotalSeconds, 2)

            $out = Join-Path $WorkRoot "reports/$($repo.name)"
            $sw = [Diagnostics.Stopwatch]::StartNew()
            $result.dry_run_exit = Invoke-LoggedCommand -FilePath "cervomut" -Arguments @("run", $repo.target, "--dry-run", "--policy", "ci-fast", "--max-mutants", "$MaxMutants", "--workers", "$Workers", "--out", $out) -WorkingDirectory $repoDir -LogPath (Join-Path $WorkRoot "$($repo.name)-dry-run.log") -TimeoutSeconds $DryRunTimeoutSeconds
            $sw.Stop()
            $result.dry_run_seconds = [math]::Round($sw.Elapsed.TotalSeconds, 2)

            if ($RunMutation) {
                $sw = [Diagnostics.Stopwatch]::StartNew()
                $result.mutation_exit = Invoke-LoggedCommand -FilePath "cervomut" -Arguments @("run", $repo.target, "--policy", "ci-balanced", "--max-mutants", "$MaxMutants", "--workers", "$Workers", "--out", $out) -WorkingDirectory $repoDir -LogPath (Join-Path $WorkRoot "$($repo.name)-mutation.log") -TimeoutSeconds $MutationTimeoutSeconds
                $sw.Stop()
                $result.mutation_seconds = [math]::Round($sw.Elapsed.TotalSeconds, 2)
                $report = Join-Path $out "mutation-report.json"
                if (Test-Path -LiteralPath $report) {
                    $json = Get-Content -LiteralPath $report -Raw | ConvertFrom-Json
                    $result.mutants = $json.summary.total
                    $result.killed = $json.summary.killed
                    $result.survived = $json.summary.survived
                    $result.not_covered = $json.summary.not_covered
                    $result.score = [math]::Round([double]$json.summary.score, 2)
                }
            }
        } finally {
            Pop-Location
        }
    } catch {
        $result.notes = $_.Exception.Message
    }
    $result.elapsed_seconds = [math]::Round(((Get-Date) - $started).TotalSeconds, 2)
    $results += [pscustomobject]$result
}

$summaryPath = Join-Path $WorkRoot "summary.json"
$results | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $summaryPath
$results | Format-Table -AutoSize
Write-Host "Summary: $summaryPath"
