# subagent_gate broken in this env; delegation protocol

## Context
Goal-driven test-writing waves in gomuks. subagent_gate tool failed 3× ("pre-launch") for agents that work fine via direct async `subagent` calls.

## Cause
Gate spawns sync/foreground runners per task (4 tasks × quorum 4) — children die during spawn validation regardless of agent name. Also `requiredSuccesses: 4` + `maxAttemptsPerTask: 1-2` = unwinnable.

## Solutions
- Never use subagent_gate for batch dispatch here; use `subagent {async:true, tasks:[full task text per child]}`.
- Disjoint file ownership per child. Dispatch ledger label→files.
- No placeholders. Fresh labeled runs instead of interrupt/resume-by-index on parallel runs.
- Child "failed" state ≠ work failed — verify by artifacts (vitest run + coverage).
- Revive via resume with concrete next-step message; children also die on connection errors mid-run.

Ref: /tmp/pi-subagents-uid-1000/async-subagent-runs/a98e45f3-9140-41f4-9567-43a1a5bd53e4/
