# payment-request-api/scripts/run-standardize-code.ps1
$ErrorActionPreference = "Stop"

Write-Host "Formatting code with gofmt..." -ForegroundColor Cyan
gofmt -w .

Write-Host "Organizing imports with goimports..." -ForegroundColor Cyan
goimports -w .

Write-Host "Running go vet..." -ForegroundColor Cyan
go vet ./...

Write-Host "Running golangci-lint..." -ForegroundColor Cyan
golangci-lint run --config=../.golangci.yml

Write-Host "Running unit tests..." -ForegroundColor Cyan
go test ./...

Write-Host "Code successfully standardized and validated!" -ForegroundColor Green
