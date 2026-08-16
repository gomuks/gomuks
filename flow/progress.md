# Progress ledger

## 2026-08-16 — coverage goal (msw3xvko)

- Baseline: web/ had 0 tests. Scope = vitest v8 coverage, target >=80%/file × 4 categories (lines/statements/functions/branches), commit+push each +20% overall.
- Wave 1 (4 delegates): util/* + ui/keybindings.ts → 20 files, 354 tests, 0 fail.
  - Commit `5141d336` pushed (scoped stmts 59.68%, branches 51.33%, funcs 63.09%, lines 59.21%).
- Wave 2 (3 delegates, disjoint dirs):
  - api/types: in flight (commands.ts was 50.74%, finishing prefs/fake/hitypes/mxtypes)
  - api/net: in flight (wsclient 1 fail left — cross-realm ArrayBuffer root cause found)
  - statestore + ui/util: DONE — 15 files, 347 tests, all files >=80%.
- New scope baseline after config include expansion: 14.72% stmts.

## Protocol (from diagnosis scout)
- subagent_gate broken in this env (pre-launch fails) → use async subagent tasks[] only
- disjoint file ownership per child, dispatch ledger
- no placeholder tasks, no interrupt+resume-by-index on parallel runs
- ignore 60s idle signals; check >=5min intervals; judge by artifacts not run state
