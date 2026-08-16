#!/usr/bin/env bash
# deploy-local.sh — Build gomuks daemon and install to ~/.local/bin/<BIN_NAME>.
# Default target: gomuks-dev (NOT gomuks — prod bins untouched by default).
#
# Mirrors ../pi-plugins deploy pattern (simplified): lockfile guard, atomic
# install, .bak rollback, deploy manifest, smoke test.
#
# Config parity: installed binary inherits env unchanged → reads prod config
# (~/.config/gomuks), data (~/.local/share/gomuks) exactly like prod.
# To isolate instead: run with GOMUKS_ROOT=~/.local/share/gomuks-dev gomuks-dev
#
# Env:
#   BIN_NAME     target bin name (default: gomuks-dev)
#   BIN_DIR      target dir (default: ~/.local/bin)
#   DEPLOY_FULL  1 = full build ./build.sh (web assets; needs web npm ci)
#                0/empty = ./build-noweb.sh (daemon only, fast)
#
# Usage:
#   bash ops/deploy/deploy-local.sh            # deploy gomuks-dev
#   bash ops/deploy/deploy-local.sh --status   # show deployed manifests
set -euo pipefail

BIN_NAME="${BIN_NAME:-gomuks-dev}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
LOCK_BASE="${XDG_RUNTIME_DIR:-/tmp}/gomuks-deploy-locks"

if [ "${1:-}" = "--status" ]; then
	for m in "$BIN_DIR"/*.manifest.json; do
		[ -f "$m" ] || { echo "no deploys"; exit 0; }
		echo "== $m"
		cat "$m"
	done
	exit 0
fi

ROOT="$(git rev-parse --show-toplevel)"

# ── Lockfile guard (mkdir-atomic, PID stale detection) ──
mkdir -p "$LOCK_BASE"
LOCK="$LOCK_BASE/$BIN_NAME"
if ! mkdir "$LOCK" 2>/dev/null; then
	owner_pid="$(cat "$LOCK/pid" 2>/dev/null || true)"
	if [ -n "$owner_pid" ] && ! kill -0 "$owner_pid" 2>/dev/null; then
		echo "⚠ stale lock (pid $owner_pid dead) — reclaiming"
		rm -rf "$LOCK"
		mkdir "$LOCK" || { echo "❌ lock race on $BIN_NAME"; exit 1; }
	else
		echo "❌ deploy lock held for $BIN_NAME (pid ${owner_pid:-unknown})"
		exit 1
	fi
fi
echo $$ >"$LOCK/pid"
release_lock() { rm -rf "$LOCK"; }
trap release_lock EXIT

echo "=== gomuks deploy: $BIN_NAME → $BIN_DIR ==="

# ── Build ──
cd "$ROOT"
if [ "${DEPLOY_FULL:-0}" = "1" ]; then
	echo "-- full build (web + daemon)"
	./build.sh
else
	echo "-- noweb build (daemon only)"
	./build-noweb.sh
fi
[ -x ./gomuks ] || { echo "❌ build output ./gomuks missing"; exit 1; }

# ── Backup previous ──
if [ -e "$BIN_DIR/$BIN_NAME" ]; then
	cp "$BIN_DIR/$BIN_NAME" "$BIN_DIR/$BIN_NAME.bak"
	echo "-- previous version → $BIN_NAME.bak"
fi

# ── Atomic install ──
install -m 0755 ./gomuks "$BIN_DIR/.$BIN_NAME.tmp"
mv "$BIN_DIR/.$BIN_NAME.tmp" "$BIN_DIR/$BIN_NAME"

# ── Manifest ──
commit="$(git rev-parse HEAD)"
branch="$(git rev-parse --abbrev-ref HEAD)"
describe="$(git describe --tags --always 2>/dev/null || echo unknown)"
ts="$(date -Iseconds)"
cat >"$BIN_DIR/$BIN_NAME.manifest.json" <<EOF
{
  "bin": "$BIN_NAME",
  "commit": "$commit",
  "branch": "$branch",
  "describe": "$describe",
  "deployedAt": "$ts",
  "full": "${DEPLOY_FULL:-0}"
}
EOF

# ── Smoke test ──
ver="$("$BIN_DIR/$BIN_NAME" --version)"
echo "$ver" | grep -q "${commit:0:7}" || echo "⚠ version string lacks current commit"
echo "✅ deployed $BIN_NAME: $ver"
