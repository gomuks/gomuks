$env:GOOS = "windows"
$env:GOARCH = $null
$env:CGO_ENABLED = "1"

if ($env:CC) {
    $env:PATH = "$env:PATH;$(Split-Path (Resolve-Path $env:CC))"
    $env:CC = $null
}

$versionPkg = "go.mau.fi/gomuks/version"
$gitCommit = git rev-parse HEAD
$gitTag = (git describe --exact-match --tags 2>$null) ?? ""
$buildTime = Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz'
$mautrixLine = Get-Content "$PSScriptRoot\go.mod" | Where-Object { $_ -match 'maunium\.net/go/mautrix\s' } | Select-Object -First 1
$mautrixVersion = ($mautrixLine -split '\s+')[1]

$ldflags = "-s -w -X '${versionPkg}.Tag=$gitTag' -X '${versionPkg}.Commit=$gitCommit' -X '${versionPkg}.BuildTime=$buildTime' -X 'maunium.net/go/mautrix.GoModVersion=$mautrixVersion'"
$baseTags = "goolm"
$tags = if ($env:GO_BUILD_TAGS) { "$baseTags $env:GO_BUILD_TAGS" } else { $baseTags }

go build -ldflags $ldflags -tags $tags @args ./cmd/gomuks-terminal
