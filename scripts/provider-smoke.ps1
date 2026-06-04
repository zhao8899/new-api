param(
    [Parameter(Mandatory = $true)]
    [string]$BaseUrl,
    [Parameter(Mandatory = $true)]
    [string]$ApiKey,
    [string]$Model = "gpt-4o-mini",
    [int]$TimeoutSec = 60,
    [switch]$SkipChat
)

$ErrorActionPreference = "Stop"
$base = $BaseUrl.TrimEnd("/")
$headers = @{
    Authorization = "Bearer $ApiKey"
    "Content-Type" = "application/json"
}

function Invoke-SmokeRequest {
    param(
        [string]$Method,
        [string]$Uri,
        [object]$Body = $null
    )

    $started = Get-Date
    $params = @{
        Method = $Method
        Uri = $Uri
        Headers = $headers
        TimeoutSec = $TimeoutSec
    }
    if ($null -ne $Body) {
        $params.Body = ($Body | ConvertTo-Json -Depth 8)
    }
    $result = Invoke-RestMethod @params
    $elapsed = [math]::Round(((Get-Date) - $started).TotalMilliseconds, 0)
    return @{ Result = $result; ElapsedMs = $elapsed }
}

Write-Host "Provider smoke: $base"

$models = Invoke-SmokeRequest -Method GET -Uri "$base/v1/models"
if ($null -eq $models.Result) {
    throw "Models endpoint returned an empty response"
}
Write-Host "PASS /v1/models $($models.ElapsedMs)ms"

if (-not $SkipChat) {
    $body = @{
        model = $Model
        messages = @(
            @{ role = "user"; content = "Return exactly: provider-smoke-ok" }
        )
        max_tokens = 16
        temperature = 0
    }
    $chat = Invoke-SmokeRequest -Method POST -Uri "$base/v1/chat/completions" -Body $body
    $content = $chat.Result.choices[0].message.content
    if ([string]::IsNullOrWhiteSpace($content)) {
        throw "Chat completion returned empty content"
    }
    Write-Host "PASS /v1/chat/completions $($chat.ElapsedMs)ms model=$Model"
}

Write-Host "Provider smoke passed"
