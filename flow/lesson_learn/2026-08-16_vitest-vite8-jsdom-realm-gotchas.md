# vitest+vite8 config + jsdom realm gotchas

## Context
gomuks web test bootstrap; vitest 4 + vite 8.2.1 (rolldown), jsdom env.

## Cause
1. `defineConfig` from "vitest/config" breaks: rolldown config bundler can't externalize transitive `node:url` ("Cannot find package 'node'"). Fix: plain object export, zero bundlable imports.
2. v8 coverage AST chokes on non-TS in include globs (json/go/css) → PARSE_ERROR. Fix: explicit coverage.exclude.
3. Cross-realm ArrayBuffer: `Buffer.buffer.slice()` fails `instanceof ArrayBuffer` under jsdom. Fix: `new ArrayBuffer()` fresh in same realm.
4. vi.fn arrow mocks break `new Audio()` — use constructible function returning object.
5. Module-level side effects (`new Audio` at import) → stub globals BEFORE import; use beforeAll import or vi.resetModules.

Ref: web/vitest.config.ts header comment, web/src/util/sound.test.ts
