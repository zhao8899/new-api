param(
    [string]$BaseUrl = "http://127.0.0.1:3000",
    [string]$Path = "/api/status",
    [ValidateRange(1, 100000)]
    [int]$Requests = 100,
    [ValidateRange(1, 512)]
    [int]$Concurrency = 10,
    [ValidateSet("GET", "POST")]
    [string]$Method = "GET",
    [string]$ApiKey = "",
    [string]$Body = "",
    [int]$TimeoutSec = 30
)

$ErrorActionPreference = "Stop"
$target = "$($BaseUrl.TrimEnd('/'))$Path"
$headers = @{}
if ($ApiKey -ne "") {
    $headers.Authorization = "Bearer $ApiKey"
}
if ($Body -ne "") {
    $headers["Content-Type"] = "application/json"
}

Write-Host "Load test target=$target requests=$Requests concurrency=$Concurrency method=$Method"

$jobs = @()
$results = New-Object System.Collections.ArrayList
$startedAt = Get-Date

for ($i = 0; $i -lt $Requests; $i++) {
    while (($jobs | Where-Object { $_.State -eq "Running" }).Count -ge $Concurrency) {
        $done = Wait-Job -Job $jobs -Any -Timeout 1
        if ($done) {
            foreach ($job in @($done)) {
                [void]$results.Add((Receive-Job -Job $job))
                Remove-Job -Job $job
                $jobs = @($jobs | Where-Object { $_.Id -ne $job.Id })
            }
        }
    }

    $jobs += Start-Job -ScriptBlock {
        param($Method, $Uri, $Headers, $Body, $TimeoutSec)
        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        try {
            $params = @{
                Method = $Method
                Uri = $Uri
                Headers = $Headers
                TimeoutSec = $TimeoutSec
                UseBasicParsing = $true
            }
            if ($Body -ne "") {
                $params.Body = $Body
            }
            $response = Invoke-WebRequest @params
            $sw.Stop()
            [pscustomobject]@{
                Ok = $response.StatusCode -ge 200 -and $response.StatusCode -lt 500
                Status = $response.StatusCode
                Ms = $sw.ElapsedMilliseconds
                Error = ""
            }
        } catch {
            $sw.Stop()
            $status = 0
            if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
                $status = [int]$_.Exception.Response.StatusCode
            }
            [pscustomobject]@{
                Ok = $false
                Status = $status
                Ms = $sw.ElapsedMilliseconds
                Error = $_.Exception.Message
            }
        }
    } -ArgumentList $Method, $target, $headers, $Body, $TimeoutSec
}

while ($jobs.Count -gt 0) {
    $done = Wait-Job -Job $jobs -Any
    foreach ($job in @($done)) {
        [void]$results.Add((Receive-Job -Job $job))
        Remove-Job -Job $job
        $jobs = @($jobs | Where-Object { $_.Id -ne $job.Id })
    }
}

$duration = ((Get-Date) - $startedAt).TotalSeconds
$latencies = @($results | ForEach-Object { [int64]$_.Ms } | Sort-Object)
$ok = @($results | Where-Object { $_.Ok }).Count
$failed = $results.Count - $ok

function Percentile {
    param([int64[]]$Values, [double]$P)
    if ($Values.Count -eq 0) { return 0 }
    $index = [math]::Ceiling(($P / 100) * $Values.Count) - 1
    $index = [math]::Max(0, [math]::Min($index, $Values.Count - 1))
    return $Values[$index]
}

Write-Host "Requests: $($results.Count)"
Write-Host "Success:  $ok"
Write-Host "Failed:   $failed"
Write-Host ("RPS:      {0:n2}" -f ($results.Count / [math]::Max($duration, 0.001)))
Write-Host "p50:      $(Percentile $latencies 50)ms"
Write-Host "p95:      $(Percentile $latencies 95)ms"
Write-Host "p99:      $(Percentile $latencies 99)ms"

if ($failed -gt 0) {
    $results | Where-Object { -not $_.Ok } | Select-Object -First 5 | Format-Table -AutoSize
    exit 1
}
