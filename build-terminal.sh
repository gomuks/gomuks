#!/bin/sh
BINARY_NAME=gomuks-terminal MAU_VERSION_PACKAGE=go.mau.fi/gomuks/version go tool maubuild -tags "$GO_BUILD_TAGS" "$@"
