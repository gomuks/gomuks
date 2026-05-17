#!/usr/bin/env bash
cd web > /dev/null
if [[ -f "./node_modules/.bin/eslint" ]]; then
	ARGS=("$@")
	ARGS=("${ARGS[@]/#web\// }")
	ARGS=("${ARGS[@]/#desktop\//../desktop/}")
	./node_modules/.bin/eslint --fix ${ARGS[@]}
fi
