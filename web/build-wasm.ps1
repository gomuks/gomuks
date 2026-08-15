$goModPath = "$PSScriptRoot\..\go.mod"
$mautrixLine = Get-Content $goModPath | Where-Object { $_ -match 'maunium\.net/go/mautrix\s' } | Select-Object -First 1
$MAUTRIX_VERSION = ($mautrixLine -split '\s+')[1]

$savedGOOS = $env:GOOS
$savedGOARCH = $env:GOARCH
$savedCGO = $env:CGO_ENABLED
try {
    $env:GOOS = "js"
    $env:GOARCH = "wasm"
    $env:CGO_ENABLED = "0"

    $tag = (git describe --exact-match --tags 2>$null) ?? ""
    $commit = git rev-parse HEAD
    $buildTime = Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz'

    $ldflags = "-X go.mau.fi/gomuks/version.Tag=$tag -X go.mau.fi/gomuks/version.Commit=$commit -X go.mau.fi/gomuks/version.BuildTime=$buildTime -X maunium.net/go/mautrix.GoModVersion=$MAUTRIX_VERSION"
    $tags = if ($env:GO_BUILD_TAGS) { "goolm sqlite_fts5 $env:GO_BUILD_TAGS" } else { "goolm sqlite_fts5" }
    $output = "$PSScriptRoot\src\api\wasm\_gomuks.wasm"
    $cmdPath = "$PSScriptRoot\..\cmd\wasmuks"

    go build -ldflags $ldflags -o $output -tags $tags $cmdPath
    $wasmExitCode = $LASTEXITCODE
} finally {
    $env:GOOS = $savedGOOS
    $env:GOARCH = $savedGOARCH
    $env:CGO_ENABLED = $savedCGO
}
if ($wasmExitCode -ne 0) { exit 2 }
