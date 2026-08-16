#!/usr/bin/env bash
# Extract per-file coverage gaps from vitest json-summary. Output: ranked gap list.
set -euo pipefail
cd "$(dirname "$0")/../web"
npx vitest --run --coverage --coverage.reporter=json-summary >/dev/null 2>&1 || true
node -e '
const s = require("./coverage/coverage-summary.json")
const rows = []
for (const [file, c] of Object.entries(s)) {
	if (file === "total") continue
	const pct = k => c[k].pct
	const min = Math.min(pct("lines"), pct("statements"), pct("functions"), pct("branches"))
	rows.push({ file: file.replace(/^.*web\/src\//, "src/"), min, lines: pct("lines"), branches: pct("branches") })
}
rows.sort((a, b) => a.min - b.min)
for (const r of rows) {
	if (r.min < 80) console.log(`${String(r.min).padStart(6)}%  lines=${String(r.lines).padStart(6)} branches=${String(r.branches).padStart(6)}  ${r.file}`)
}
const t = s.total
console.log(`--- TOTAL lines=${t.lines.pct}% stmts=${t.statements.pct}% funcs=${t.functions.pct}% branches=${t.branches.pct}%`)
'
