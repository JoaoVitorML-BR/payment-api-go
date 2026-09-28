# Run standardization for all services
$ErrorActionPreference = "Stop"

Write-Host "=== Standardizing payment-consumer ===" -ForegroundColor Yellow
Set-Location payment-consumer
.\scripts\run-standardize-code.ps1

Write-Host "`n=== Standardizing payment-request-api ===" -ForegroundColor Yellow
Set-Location ..\payment-request-api
.\scripts\run-standardize-code.ps1

Set-Location ..
Write-Host "`nAll services standardized and verified successfully!" -ForegroundColor Green
