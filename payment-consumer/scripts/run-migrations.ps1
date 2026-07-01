param(
    [string]$ComposeService = "postgres"
)

$ErrorActionPreference = "Stop"

docker compose up -d $ComposeService | Out-Null

$DatabaseUser = (& docker compose exec -T $ComposeService printenv POSTGRES_USER).Trim()
$DatabaseName = (& docker compose exec -T $ComposeService printenv POSTGRES_DB).Trim()
$DatabasePassword = (& docker compose exec -T $ComposeService printenv POSTGRES_PASSWORD).Trim()

if (-not $DatabaseUser -or -not $DatabaseName) {
    throw "Could not read POSTGRES_USER or POSTGRES_DB from the running container."
}

if (-not $DatabasePassword) {
    throw "Could not read POSTGRES_PASSWORD from the running container."
}

$migrationsDir = Join-Path $PSScriptRoot "..\internal\infra\database\migrations"
$migrationFiles = Get-ChildItem -Path $migrationsDir -Filter "*.sql" | Sort-Object Name

foreach ($file in $migrationFiles) {
    Write-Host "Applying $($file.Name)..."
    $psqlCommand = "set -e; PGPASSWORD='$DatabasePassword' psql -U '$DatabaseUser' -d '$DatabaseName' -v ON_ERROR_STOP=1 -f -"
    Get-Content -Raw -Path $file.FullName |
        & docker compose exec -T $ComposeService sh -lc $psqlCommand

    if ($LASTEXITCODE -ne 0) {
        throw "Migration $($file.Name) failed with exit code $LASTEXITCODE"
    }
}

Write-Host "All migrations applied."
