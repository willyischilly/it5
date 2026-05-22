$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
Set-Location $root

Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue |
    ForEach-Object { Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue }

Write-Host "Starting http://localhost:8080 ..." -ForegroundColor Cyan
if (Test-Path "bin\planner-backend.exe") {
    & ".\bin\planner-backend.exe"
} else {
    Write-Host "bin not found, using: go run ./cmd/main.go" -ForegroundColor Yellow
    go run ./cmd/main.go
}
