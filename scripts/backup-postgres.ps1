param(
    [string]$ContainerName = "postgres",
    [string]$Database = "new-api",
    [string]$User = "root",
    [string]$BackupDir = ".\backup",
    [string]$DumpPath = "",
    [switch]$RestoreDrill,
    [string]$RestoreDatabase = "new-api-restore"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "docker command not found"
}

if (-not (Test-Path $BackupDir)) {
    New-Item -ItemType Directory -Path $BackupDir | Out-Null
}

if ($DumpPath -eq "") {
    $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $DumpPath = Join-Path $BackupDir "$Database-$stamp.dump"
}

$containerDump = "/tmp/$(Split-Path -Leaf $DumpPath)"

Write-Host "Creating PostgreSQL backup container=$ContainerName database=$Database"
docker exec $ContainerName pg_dump --username $User --format custom --no-owner --no-privileges --file $containerDump $Database
docker cp "$ContainerName`:$containerDump" $DumpPath
docker exec $ContainerName rm -f $containerDump | Out-Null

if (-not (Test-Path $DumpPath)) {
    throw "Backup file was not created: $DumpPath"
}

$size = (Get-Item $DumpPath).Length
Write-Host "Backup written: $DumpPath bytes=$size"

if ($RestoreDrill) {
    Write-Host "Running restore drill database=$RestoreDatabase"
    docker exec $ContainerName dropdb --username $User --if-exists $RestoreDatabase | Out-Null
    docker exec $ContainerName createdb --username $User $RestoreDatabase
    docker cp $DumpPath "$ContainerName`:$containerDump"
    docker exec $ContainerName pg_restore --username $User --dbname $RestoreDatabase --clean --if-exists --no-owner --no-privileges $containerDump
    docker exec $ContainerName rm -f $containerDump | Out-Null
    Write-Host "Restore drill passed: $RestoreDatabase"
}
