param(
    [string]$Manifest = "docs/evaluations/go-repo-pool-40.json",
    [string]$WorkRoot = "$env:TEMP/cervomut-go-pool-40",
    [string]$OutputRoot = "$env:TEMP/cervomut-tool-comparison-12",
    [string[]]$Names = @("cobra", "pflag", "moby", "hugo", "prometheus", "terraform", "grpc-go", "echo", "logrus", "validator", "decimal", "gjson"),
    [string[]]$Tools = @("cervomut", "gremlins", "gomu", "go-mutesting"),
    [int]$Workers = 2,
    [int]$TimeoutSeconds = 600,
    [switch]$Resume,
    [string]$CervoMutant = "$env:TEMP/cervomut-pool.exe",
    [string]$Gremlins = "$env:TEMP/cervomut-study-cobra/tools/gremlins.exe",
    [string]$Gomu = "$env:TEMP/cervomut-study-cobra/tools/gomu-patched.exe",
    [string]$GoMutesting = "$env:TEMP/cervomut-study-cobra/tools/go-mutesting-patched.exe"
)

$ErrorActionPreference = "Stop"

$manifestData = Get-Content -LiteralPath $Manifest -Raw | ConvertFrom-Json
$wanted = @{}
foreach ($name in $Names) {
    $wanted[$name] = $true
}
$repos = @($manifestData.repos | Where-Object { $wanted.ContainsKey($_.name) })
$wantedTools = @{}
foreach ($tool in $Tools) {
    $wantedTools[$tool] = $true
}

New-Item -ItemType Directory -Path $OutputRoot -Force | Out-Null

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
    $proc.Refresh()
    return [int]$proc.ExitCode
}

function Read-CervoReport($path) {
    if (!(Test-Path -LiteralPath $path)) { return @{} }
    $j = Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
    return @{
        total = $j.summary.total
        killed = $j.summary.killed
        survived = $j.summary.survived
        not_covered = $j.summary.not_covered
        errors = $j.summary.compile_error
        timed_out = $j.summary.timed_out
        score = [math]::Round([double]$j.summary.score, 2)
    }
}

function Read-GremlinsReport($path) {
    if (!(Test-Path -LiteralPath $path)) { return @{} }
    $j = Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
    return @{
        total = $j.mutants_total
        killed = $j.mutants_killed
        survived = $j.mutants_lived
        not_covered = $j.mutants_not_covered
        errors = 0
        timed_out = (($j.files | ForEach-Object { $_.mutations } | Where-Object { $_.status -eq "TIMED OUT" }).Count)
        score = [math]::Round([double]$j.test_efficacy, 2)
    }
}

function Read-GomuReport($path) {
    if (!(Test-Path -LiteralPath $path)) { return @{} }
    $j = Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
    $groups = @{}
    foreach ($g in ($j.results | Group-Object status)) {
        $groups[$g.Name] = $g.Count
    }
    $killed = 0
    $survived = 0
    $errors = 0
    $notViable = 0
    if ($groups.ContainsKey("KILLED")) { $killed = [int]$groups["KILLED"] }
    if ($groups.ContainsKey("SURVIVED")) { $survived = [int]$groups["SURVIVED"] }
    if ($groups.ContainsKey("ERROR")) { $errors = [int]$groups["ERROR"] }
    if ($groups.ContainsKey("NOT_VIABLE")) { $notViable = [int]$groups["NOT_VIABLE"] }
    $denom = $killed + $survived
    $score = 0.0
    if ($denom -gt 0) { $score = [math]::Round(($killed / $denom) * 100, 2) }
    return @{
        total = $j.totalMutants
        killed = $killed
        survived = $survived
        not_covered = 0
        not_viable = $notViable
        errors = $errors
        timed_out = 0
        score = $score
    }
}

function Read-GoMutestingReport($path) {
    if (!(Test-Path -LiteralPath $path)) { return @{} }
    $j = Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
    return @{
        total = $j.stats.totalMutantsCount
        killed = $j.stats.killedCount
        survived = $j.stats.escapedCount
        not_covered = $j.stats.notCoveredCount
        errors = $j.stats.errorCount
        timed_out = $j.stats.timeOutCount
        score = [math]::Round([double]$j.stats.msi * 100, 2)
    }
}

$results = @()
$summaryPath = Join-Path $OutputRoot "summary.json"
if ($Resume -and (Test-Path -LiteralPath $summaryPath)) {
    $loaded = @(Get-Content -LiteralPath $summaryPath -Raw | ConvertFrom-Json)
    foreach ($item in $loaded) {
        if ($item.PSObject.Properties.Name -contains "value") {
            $results += @($item.value)
        } elseif ($item.PSObject.Properties.Name -contains "repo") {
            $results += $item
        }
    }
}

function Has-Result($results, $repo, $tool) {
    return @($results | Where-Object { $_.repo -eq $repo -and $_.tool -eq $tool }).Count -gt 0
}

foreach ($repo in $repos) {
    $repoDir = Join-Path $WorkRoot $repo.name
    if (!(Test-Path -LiteralPath $repoDir)) {
        $results += [pscustomobject]@{ repo = $repo.name; tool = "all"; exit = 127; seconds = 0; note = "repo checkout missing" }
        continue
    }
    $repoOut = Join-Path $OutputRoot $repo.name
    New-Item -ItemType Directory -Path $repoOut -Force | Out-Null
    $tools = @(
        @{ name = "cervomut"; exe = $CervoMutant; args = @("run", $repo.target, "--profile", "gremlins-compatible", "--isolation", "overlay", "--workers", "$Workers", "--out", (Join-Path $repoOut "cervomut")); report = Join-Path $repoOut "cervomut/mutation-report.json"; parser = "cervo" },
        @{ name = "gremlins"; exe = $Gremlins; args = @("unleash", $repo.target, "--workers", "$Workers", "--threshold-efficacy", "0", "--threshold-mcover", "0", "--output", (Join-Path $repoOut "gremlins.json")); report = Join-Path $repoOut "gremlins.json"; parser = "gremlins" },
        @{ name = "gomu"; exe = $Gomu; args = @("run", $repo.target, "--workers", "$Workers", "--timeout", "30", "--threshold", "0", "--fail-on-gate=false", "--output", "json"); report = Join-Path $repoDir "mutation-report.json"; parser = "gomu" },
        @{ name = "go-mutesting"; exe = $GoMutesting; args = @("/noop", "/quiet", "/no-diffs", "/logger-summary-json", "/logger-agentic-json", "/exec-timeout:30", "/workers:$Workers", $repo.target); report = Join-Path $repoDir "report.json"; parser = "go-mutesting" }
    )
    $tools = @($tools | Where-Object { $wantedTools.ContainsKey($_["name"]) })
    foreach ($tool in $tools) {
        if ($Resume -and (Has-Result $results $repo.name $tool["name"])) {
            continue
        }
        Remove-Item -LiteralPath $tool["report"] -Force -ErrorAction SilentlyContinue
        $log = Join-Path $repoOut "$($tool["name"]).log"
        $sw = [Diagnostics.Stopwatch]::StartNew()
        $exit = Invoke-LoggedCommand -FilePath $tool["exe"] -Arguments $tool["args"] -WorkingDirectory $repoDir -LogPath $log -TimeoutSeconds $TimeoutSeconds
        $sw.Stop()
        $metrics = @{}
        switch ($tool["parser"]) {
            "cervo" { $metrics = Read-CervoReport $tool["report"] }
            "gremlins" { $metrics = Read-GremlinsReport $tool["report"] }
            "gomu" {
                if (Test-Path -LiteralPath $tool["report"]) {
                    Copy-Item -LiteralPath $tool["report"] -Destination (Join-Path $repoOut "gomu-mutation-report.json") -Force
                    $metrics = Read-GomuReport $tool["report"]
                }
            }
            "go-mutesting" {
                if (Test-Path -LiteralPath $tool["report"]) {
                    Copy-Item -LiteralPath $tool["report"] -Destination (Join-Path $repoOut "go-mutesting-report.json") -Force
                    $metrics = Read-GoMutestingReport $tool["report"]
                }
            }
        }
        $results += [pscustomobject]([ordered]@{
            repo = $repo.name
            target = $repo.target
            lane = $repo.lane
            domain = $repo.domain
            tool = $tool["name"]
            exit = $exit
            seconds = [math]::Round($sw.Elapsed.TotalSeconds, 2)
            total = $metrics.total
            killed = $metrics.killed
            survived = $metrics.survived
            not_covered = $metrics.not_covered
            not_viable = $metrics.not_viable
            errors = $metrics.errors
            timed_out = $metrics.timed_out
            score = $metrics.score
            log = $log
        })
        $results | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $summaryPath
    }
}

$results | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $summaryPath
$results | Format-Table -AutoSize
Write-Host "Summary: $summaryPath"
