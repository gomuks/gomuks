$env:GOOS = "windows"
$env:GOARCH = $null
$env:CGO_ENABLED = "1"

# Go 1.26 on Windows doesn't use CC when set to an absolute path; add gcc's
# directory to PATH instead. Append (not prepend) to avoid shadowing the Go toolchain.
if ($env:CC) {
    $env:PATH = "$env:PATH;$(Split-Path (Resolve-Path $env:CC))"
    $env:CC = $null
}

$distDir = "$PSScriptRoot\web\dist"
New-Item -ItemType Directory -Force -Path $distDir | Out-Null
if (-not (Get-ChildItem -Path $distDir -Force -ErrorAction SilentlyContinue)) {
    New-Item -ItemType File -Path "$distDir\empty" | Out-Null
}

$versionPkg = "go.mau.fi/gomuks/version"
$gitCommit = git rev-parse HEAD
$gitTag = (git describe --exact-match --tags 2>$null) ?? ""
$buildTime = Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz'
$mautrixLine = Get-Content "$PSScriptRoot\go.mod" | Where-Object { $_ -match 'maunium\.net/go/mautrix\s' } | Select-Object -First 1
$mautrixVersion = ($mautrixLine -split '\s+')[1]

$ldflags = "-s -w -X '${versionPkg}.Tag=$gitTag' -X '${versionPkg}.Commit=$gitCommit' -X '${versionPkg}.BuildTime=$buildTime' -X 'maunium.net/go/mautrix.GoModVersion=$mautrixVersion'"
$tags = if ($env:GO_BUILD_TAGS) { "sqlite_fts5 goolm $env:GO_BUILD_TAGS" } else { "sqlite_fts5 goolm" }

go build -ldflags $ldflags -tags $tags @args ./cmd/gomuks
