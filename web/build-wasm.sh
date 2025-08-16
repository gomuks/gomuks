#!/bin/sh
export MAUTRIX_VERSION=$(cat ../go.mod | grep 'maunium.net/go/mautrix ' | head -n1 | awk '{ print $2 }')
export GOOS=js
export GOARCH=wasm
GO_LDFLAGS="-X go.mau.fi/gomuks/version.Tag=$(git describe --exact-match --tags 2>/dev/null) -X go.mau.fi/gomuks/version.Commit=$(git rev-parse HEAD) -X 'go.mau.fi/gomuks/version.BuildTime=`date -Iseconds`' -X 'maunium.net/go/mautrix.GoModVersion=$MAUTRIX_VERSION'"
go build -ldflags "$GO_LDFLAGS" -o src/api/wasm/_gomuks.wasm -tags goolm ../cmd/wasmuks || exit 2
