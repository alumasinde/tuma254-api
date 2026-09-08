param(
    [string]$BaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"

function Assert-Status {
    param([int]$Expected, [string]$Method, [string]$Path, [string]$Body = "")
    try {
        if ($Body -ne "") {
            $response = Invoke-WebRequest -Method $Method -Uri "$BaseUrl$Path" -ContentType "application/json" -Body $Body
        } else {
            $response = Invoke-WebRequest -Method $Method -Uri "$BaseUrl$Path"
        }
        if ($response.StatusCode -ne $Expected) {
            throw "$Method $Path returned $($response.StatusCode), expected $Expected"
        }
    } catch {
        if ($_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq $Expected) {
            return
        }
        throw
    }
}

Assert-Status 200 "GET" "/health"
Assert-Status 200 "GET" "/ready"
Assert-Status 401 "GET" "/api/v1/me"
Assert-Status 400 "POST" "/api/v1/auth/login" '{"email":"invalid@example.com"}'

Write-Host "API smoke checks passed."
