$env:GOOS = $null
$env:GOARCH = $null
$env:CGO_ENABLED = $null

$tempExe = Join-Path ([System.IO.Path]::GetTempPath()) "gomuks-print.exe"
go build -o $tempExe "$PSScriptRoot\..\pkg\hicli\cmdspec\print"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
try {
    & $tempExe `
        "$PSScriptRoot\src\api\types\stdcommands.json" `
        "$PSScriptRoot\src\api\types\stdcommands.d.ts" `
        "$PSScriptRoot\src\api\types\commandtestdata"
} finally {
    Remove-Item $tempExe -ErrorAction SilentlyContinue
}
