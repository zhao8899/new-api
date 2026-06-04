param(
    [string]$ImageTag = "new-api-release-smoke:local",
    [string]$Prefix = "new-api-release-smoke",
    [int]$AppPort = 3020,
    [switch]$SkipBuild,
    [switch]$KeepContainers
)

$ErrorActionPreference = "Stop"
$script:SmokeWebSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession

function Write-Step {
    param([string]$Message)
    Write-Host "[release-smoke] $Message"
}

function Invoke-JsonRequest {
    param(
        [string]$Method,
        [string]$Uri,
        [object]$Body = $null,
        [hashtable]$Headers = @{}
    )

    $params = @{
        Method      = $Method
        Uri         = $Uri
        Headers     = $Headers
        WebSession  = $script:SmokeWebSession
        TimeoutSec  = 15
        ErrorAction = "Stop"
    }
    if ($null -ne $Body) {
        $params.ContentType = "application/json"
        $params.Body = ($Body | ConvertTo-Json -Depth 20 -Compress)
    }
    return Invoke-RestMethod @params
}

function Wait-HttpReady {
    param([string]$StatusUrl)

    $deadline = (Get-Date).AddSeconds(90)
    do {
        try {
            $status = Invoke-JsonRequest -Method "GET" -Uri $StatusUrl
            if ($status.success -eq $true) {
                return $status
            }
        } catch {
            Start-Sleep -Seconds 2
        }
    } while ((Get-Date) -lt $deadline)

    throw "service did not become ready at $StatusUrl"
}

function Remove-SmokeContainers {
    param([string]$NamePrefix)

    $names = @(
        "$NamePrefix-app",
        "$NamePrefix-pg",
        "$NamePrefix-redis"
    )
    foreach ($name in $names) {
        $existing = docker ps -a --filter "name=^/$name$" --format "{{.Names}}"
        if ($existing -eq $name) {
            docker rm -f $name | Out-Null
        }
    }

    $network = docker network ls --filter "name=^$NamePrefix-net$" --format "{{.Name}}"
    if ($network -eq "$NamePrefix-net") {
        docker network rm "$NamePrefix-net" | Out-Null
    }
}

function Assert-Success {
    param(
        [object]$Response,
        [string]$Context
    )
    if ($Response.success -ne $true) {
        $message = ""
        if ($Response.PSObject.Properties.Name -contains "message") {
            $message = [string]$Response.message
        }
        throw "$Context failed: $message"
    }
}

$networkName = "$Prefix-net"
$postgresName = "$Prefix-pg"
$redisName = "$Prefix-redis"
$appName = "$Prefix-app"
$postgresPassword = "smoke-postgres-pass"
$sessionSecret = "release-smoke-session-secret-0000000000000000"
$cryptoSecret = "release-smoke-crypto-secret-000000000000000000"
$setupToken = "release-smoke-setup-token"
$rootUsername = "rootsmoke"
$rootPassword = "RootSmokePass123!"
$baseUrl = "http://127.0.0.1:$AppPort"

try {
    Write-Step "checking Docker availability"
    docker info | Out-Null

    if (-not $SkipBuild) {
        Write-Step "building image $ImageTag"
        docker build -f Dockerfile -t $ImageTag . | Out-Host
        if ($LASTEXITCODE -ne 0) {
            throw "docker build failed with exit code $LASTEXITCODE"
        }
    }

    Write-Step "cleaning prior smoke containers"
    Remove-SmokeContainers -NamePrefix $Prefix

    Write-Step "creating isolated Docker network $networkName"
    docker network create $networkName | Out-Null

    Write-Step "starting PostgreSQL"
    docker run -d `
        --name $postgresName `
        --network $networkName `
        -e POSTGRES_USER=root `
        -e POSTGRES_PASSWORD=$postgresPassword `
        -e POSTGRES_DB=new-api `
        postgres:15-alpine | Out-Null

    Write-Step "starting Redis"
    docker run -d `
        --name $redisName `
        --network $networkName `
        redis:7-alpine | Out-Null

    Write-Step "waiting for PostgreSQL"
    $pgDeadline = (Get-Date).AddSeconds(60)
    do {
        docker exec $postgresName pg_isready -U root -d new-api | Out-Null
        if ($LASTEXITCODE -eq 0) {
            break
        }
        Start-Sleep -Seconds 2
    } while ((Get-Date) -lt $pgDeadline)
    if ($LASTEXITCODE -ne 0) {
        throw "PostgreSQL did not become ready"
    }

    Write-Step "starting app on $baseUrl"
    $sqlDsn = "postgresql://root:$postgresPassword@${postgresName}:5432/new-api"
    $redisDsn = "redis://${redisName}:6379/0"
    docker run -d `
        --name $appName `
        --network $networkName `
        -p "$AppPort`:3000" `
        --env "SQL_DSN=$sqlDsn" `
        --env "REDIS_CONN_STRING=$redisDsn" `
        --env "SESSION_SECRET=$sessionSecret" `
        --env "CRYPTO_SECRET=$cryptoSecret" `
        --env "NEW_API_SETUP_TOKEN=$setupToken" `
        --env "TZ=Asia/Shanghai" `
        $ImageTag | Out-Null

    Write-Step "waiting for /api/status"
    $status = Wait-HttpReady -StatusUrl "$baseUrl/api/status"
    Assert-Success -Response $status -Context "/api/status"

    Write-Step "running setup flow"
    $setupStatus = Invoke-JsonRequest -Method "GET" -Uri "$baseUrl/api/setup"
    Assert-Success -Response $setupStatus -Context "GET /api/setup"
    if ($setupStatus.data.status -ne $true) {
        $setupResponse = Invoke-JsonRequest -Method "POST" -Uri "$baseUrl/api/setup" -Body @{
            username           = $rootUsername
            password           = $rootPassword
            confirmPassword    = $rootPassword
            setup_token        = $setupToken
            SelfUseModeEnabled = $false
            DemoSiteEnabled    = $false
        }
        Assert-Success -Response $setupResponse -Context "POST /api/setup"
    }

    Write-Step "logging in root user"
    $login = Invoke-JsonRequest -Method "POST" -Uri "$baseUrl/api/user/login" -Body @{
        username = $rootUsername
        password = $rootPassword
    }
    Assert-Success -Response $login -Context "POST /api/user/login"
    $userId = [string]$login.data.id
    if ([string]::IsNullOrWhiteSpace($userId)) {
        throw "login response did not include user id"
    }
    $authHeaders = @{ "New-Api-User" = $userId }

    Write-Step "checking authenticated account endpoint"
    $self = Invoke-JsonRequest -Method "GET" -Uri "$baseUrl/api/user/self" -Headers $authHeaders
    Assert-Success -Response $self -Context "GET /api/user/self"

    Write-Step "checking payment top-up info endpoint"
    $topupInfo = Invoke-JsonRequest -Method "GET" -Uri "$baseUrl/api/user/topup/info" -Headers $authHeaders
    Assert-Success -Response $topupInfo -Context "GET /api/user/topup/info"

    Write-Step "checking registry endpoints"
    $models = Invoke-JsonRequest -Method "GET" -Uri "$baseUrl/api/registry/models" -Headers $authHeaders
    Assert-Success -Response $models -Context "GET /api/registry/models"
    $providers = Invoke-JsonRequest -Method "GET" -Uri "$baseUrl/api/registry/providers" -Headers $authHeaders
    Assert-Success -Response $providers -Context "GET /api/registry/providers"

    Write-Step "checking channel health mode endpoint"
    $healthMode = Invoke-JsonRequest -Method "GET" -Uri "$baseUrl/api/channel/health/mode" -Headers $authHeaders
    Assert-Success -Response $healthMode -Context "GET /api/channel/health/mode"

    Write-Step "checking payment reconciliation endpoint"
    $now = [int][double]::Parse((Get-Date -UFormat %s))
    $reconciliationUrl = "$baseUrl/api/user/topup/reconciliation?start_time=$($now - 86400)&end_time=$now&payment_provider=epay"
    $reconciliation = Invoke-JsonRequest -Method "GET" -Uri $reconciliationUrl -Headers $authHeaders
    Assert-Success -Response $reconciliation -Context "GET /api/user/topup/reconciliation"

    Write-Step "checking frontend shell"
    $web = Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/" -TimeoutSec 15
    if ($web.StatusCode -ne 200 -or -not $web.Content.Contains("New API")) {
        throw "frontend shell did not return expected content"
    }

    Write-Step "PASS image=$ImageTag url=$baseUrl"
    if ($KeepContainers) {
        Write-Step "containers kept: $appName, $postgresName, $redisName"
    }
} finally {
    if (-not $KeepContainers) {
        Write-Step "cleaning smoke containers"
        Remove-SmokeContainers -NamePrefix $Prefix
    }
}
