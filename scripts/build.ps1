$ErrorActionPreference = "Stop"
$root = Split-Path $PSScriptRoot -Parent
Set-Location $root

New-Item -ItemType Directory -Force -Path "bin" | Out-Null

Write-Host "Go: $(go version)" -ForegroundColor Cyan
Write-Host "Building..." -ForegroundColor Cyan

go mod tidy
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o bin/planner-backend.exe ./cmd/main.go

if ($LASTEXITCODE -ne 0) { exit 1 }

Write-Host "OK: $root\bin\planner-backend.exe" -ForegroundColor Green
