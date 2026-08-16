#!/usr/bin/env bash
# deploy-local.sh — Build gomuks and install to ~/.local/bin/<BIN_NAME>.
# Default target: gomuks-dev (SERVER). Prod bins untouched by default.
#
# Modes:
#   server (default) = cmd/gomuks          — daemon, single instance per host
#   client           = cmd/gomuks-terminal — terminal UI, connect to daemon,
#                                            can deploy MULTIPLE versions side by side
#
# Mirrors ../pi-plugins deploy pattern (simplified): lockfile guard, atomic
# install, .bak rollback, deploy manifest, smoke test.
#
# Config parity: installed binary inherits env unchanged → reads prod config
# (~/.config/gomuks), data (~/.local/share/gomuks) exactly like prod.
# Isolated server run: GOMUKS_ROOT=~/.local/share/gomuks-dev gomuks-dev
#
# Env:
#   MODE         server | client (default: server)
#   BIN_NAME     target bin name (default: gomuks-dev / gomuks-terminal-dev)
#                client mode allows suffixed names (gomuks-terminal-dev2, ...)
#   BIN_DIR      target dir (default: ~/.local/bin)
#   DEPLOY_FULL  1 = full build ./build.sh (server only; web assets, needs web npm ci)
#                0/empty = daemon/terminal-only build (fast)
#
# Usage:
#   bash ops/deploy/deploy-local.sh                  # deploy server → gomuks-dev
#   MODE=client bash ops/deploy/deploy-local.sh      # deploy client → gomuks-terminal-dev
#   MODE=client BIN_NAME=gomuks-terminal-dev2 bash ops/deploy/deploy-local.sh
#   bash ops/deploy/deploy-local.sh --status         # show deployed manifests
set -euo pipefail

MODE="${MODE:-server}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
LOCK_BASE="${XDG_RUNTIME_DIR:-/tmp}/gomuks-deploy-locks"

case "$MODE" in
server)
	# Daemon = singleton per host: always keep .bak rollback.
	DEFAULT_BIN="gomuks-dev"
	KEEP_BAK=1
	;;
client)
	DEFAULT_BIN="gomuks-terminal-dev"
	KEEP_BAK=0
	;;
*)
	echo "❌ unknown MODE '$MODE' (server|client)"
	exit 2
	;;
esac
BIN_NAME="${BIN_NAME:-$DEFAULT_BIN}"

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

echo "=== gomuks deploy [$MODE]: $BIN_NAME → $BIN_DIR ==="

# ── Build ──
cd "$ROOT"
if [ "$MODE" = "server" ] && [ "${DEPLOY_FULL:-0}" = "1" ]; then
	echo "-- full build (web + daemon)"
	./build.sh
else
	echo "-- $MODE build (no web)"
	if [ "$MODE" = "client" ]; then
		./build-terminal.sh
	else
		./build-noweb.sh
	fi
fi
SRC="./gomuks-terminal"
[ "$MODE" = "server" ] && SRC="./gomuks"
[ -x "$SRC" ] || { echo "❌ build output $SRC missing"; exit 1; }

# Client multi-install → same-name overwrites NOT backed up (keep both old
# suffixed bins + manifests instead). FORCE_BAK=1 overrides.
if [ -e "$BIN_DIR/$BIN_NAME" ] && { [ "${FORCE_BAK:-0}" = "1" ] || [ "$KEEP_BAK" = "1" ]; }; then
	cp "$BIN_DIR/$BIN_NAME" "$BIN_DIR/$BIN_NAME.bak"
	echo "-- previous version → $BIN_NAME.bak"
fi

# ── Atomic install ──
install -m 0755 "$SRC" "$BIN_DIR/.$BIN_NAME.tmp"
mv "$BIN_DIR/.$BIN_NAME.tmp" "$BIN_DIR/$BIN_NAME"

# ── Manifest ──
commit="$(git rev-parse HEAD)"
branch="$(git rev-parse --abbrev-ref HEAD)"
describe="$(git describe --tags --always 2>/dev/null || echo unknown)"
ts="$(date -Iseconds)"
cat >"$BIN_DIR/$BIN_NAME.manifest.json" <<EOF
{
  "bin": "$BIN_NAME",
  "mode": "$MODE",
  "commit": "$commit",
  "branch": "$branch",
  "describe": "$describe",
  "deployedAt": "$ts",
  "full": "${DEPLOY_FULL:-0}"
}
EOF

# ── Smoke test ──
ver="$("$BIN_DIR/$BIN_NAME" --version 2>/dev/null || echo "(no --version)")"
echo "$ver" | grep -q "${commit:0:7}" || echo "⚠ version string lacks current commit"
echo "✅ deployed $BIN_NAME [$MODE]: $ver"
