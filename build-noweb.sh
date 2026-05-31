#!/usr/bin/env bash
mkdir -p web/dist/
if [[ -z "$(ls -A web/dist/)" ]]; then
	touch web/dist/empty
fi
BINARY_NAME=gomuks MAU_VERSION_PACKAGE=go.mau.fi/gomuks/version go tool maubuild -tags "sqlite_fts5 $GO_BUILD_TAGS" "$@"
