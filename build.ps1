if ( -not ( Get-Command "gcc.exe" -ErrorAction SilentlyContinue ) -and -not ( $env:CC -and ( Get-Command $env:CC -ErrorAction SilentlyContinue ) ) ) {
    Write-Error "gcc.exe not found in PATH."
    Write-Error "Please install MinGW-w64 and add it to your PATH or set the CC environment variable to the gcc executable."
    exit 1
}

& "$PSScriptRoot\web\export-go-data.ps1"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& "$PSScriptRoot\web\build-wasm.ps1"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Push-Location "$PSScriptRoot\web"
try {
    npm install
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    npm run build
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Pop-Location
}

& "$PSScriptRoot\build-noweb.ps1" @args
