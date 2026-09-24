Push-Location $PSScriptRoot
try {
    $versionPkg = "go.mau.fi/gomuks/version"
    $gitCommit = git rev-parse HEAD
    $gitTag = (git describe --exact-match --tags 2>$null) ?? ""
    $buildTime = Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz'
    $mautrixLine = Get-Content "$PSScriptRoot\..\..\go.mod" | Where-Object { $_ -match 'maunium\.net/go/mautrix\s' } | Select-Object -First 1
    $mautrixVersion = ($mautrixLine -split '\s+')[1]

    $ldflags = "-s -w -X '${versionPkg}.Tag=$gitTag' -X '${versionPkg}.Commit=$gitCommit' -X '${versionPkg}.BuildTime=$buildTime' -X 'maunium.net/go/mautrix.GoModVersion=$mautrixVersion'"

    go build -ldflags $ldflags -buildmode=c-shared -o libgomuksffi.dll @args .
    Remove-Item -Force -ErrorAction SilentlyContinue "$PSScriptRoot\libgomuksffi.h"
} finally {
    Pop-Location
}
