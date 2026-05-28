# docgen — jsoncmd reference generator

A standalone Go program that parses `pkg/hicli/jsoncmd` and emits a single
self-contained HTML file documenting every command and event. It is
**hardcoded** to that one package (per `INITIAL_PROMPT.md`), but it discovers
the actual commands by walking the AST — adding or removing a spec in
`commands.go` requires no changes here.

## Run it

From the module root:

```
go run ./pkg/hicli/jsoncmd/docgen -o jsoncmd.html
```

Flags:

- `-o` — output path (default `jsoncmd.html`)
- `-root` — module root dir (default `.`, i.e. the working directory must be
  the module root for the default to work)

## What it reads

The package walks the `jsoncmd` package looking for **package-level `var`
declarations whose value is a composite literal of one of the spec generic
types** (defined in `schema.go:specTypes`):

| Type                         | Generic shape         | Section  |
| ---------------------------- | --------------------- | -------- |
| `CommandSpec[Req, Resp]`     | request + response    | Commands |
| `CommandSpecWithoutResponse[Req]` | request only      | Commands |
| `CommandSpecWithoutRequest[Resp]` | response only     | Commands |
| `CommandSpecWithoutData`     | neither               | Commands |
| `EventSpec[Payload]`         | payload only          | Events   |

For each match, the generator pulls:

1. **On-wire name** — the value of the `Name:` field in the composite literal.
   It supports both a string literal and an identifier that resolves to a
   string constant in the same package (`extract.go:resolveNameField`).
2. **Docstring** — the `var`'s doc comment. The variable name at the start is
   replaced with `` `cmd_name` `` so the rendered doc reads naturally
   (`render.go:replaceLeadingVarName`).
3. **Request / response types** — extracted from the generic type arguments
   and rendered via the schema walker (`schema.go:renderType`).

Specs with no resolvable `Name` are silently skipped — incomplete docs are
worse than missing entries.

## How types are rendered

`schema.go` builds a `TypeRef` tree from `go/ast` expressions. Key behaviors:

- **Pointers** are unwrapped (JSON doesn't distinguish `*T` from `T` beyond
  nullability).
- **`[]byte`** is treated as a basic type (base64 JSON string), not a slice.
- **In-module named types** (under `go.mau.fi/gomuks`) are expanded inline
  *and* linked to pkg.go.dev. External types only get the link.
- **Cycles** are broken with a `visited` set keyed by `importPath.TypeName`.
  Re-entry sets `Recursive: true` and stops expanding.
- **Embedded fields** are kept and labeled. The display name is the terminal
  type ident (`embeddedDisplayName`).
- **`json` struct tags** drive field naming and `optional` markers
  (`omitempty`/`omitzero`); `json:"-"` fields are dropped.
- **Generics** (`Container[A, B]`) are handled — the base name is rendered as
  a named ref and type args are attached as `TypeArgs`.

The HTML template (`template.html`) chooses between an expandable
`<details>` block and a leaf `schema-leaf` div based on
`TypeRef.HasInlineStruct()` — anything with reachable struct content is
collapsible; basic/external-only types render inline.

## Markdown

Variable docstrings and field doc comments are run through goldmark
(`render.go:renderMarkdown`) with default extensions. The result is injected
as `template.HTML` (trusted — source is our own Go comments).

## File map

| File             | Responsibility |
| ---------------- | -------------- |
| `main.go`        | CLI entry point, wires up loader → extractor → page builder → template. |
| `loader.go`      | Parses Go packages with `go/parser`. Caches by import path; indexes types, string constants, and per-file import aliases. In-module only. |
| `extract.go`     | Walks the jsoncmd AST, matches spec types, resolves the on-wire name, and returns sorted `rawSpec`s (sorted by source position so the output mirrors source order). |
| `schema.go`      | AST → `TypeRef` tree. Field expansion, cycle handling, struct-tag parsing, pkg.go.dev linkification, markdown rendering of field docs. |
| `render.go`      | `rawSpec` → `Entry` → `Page`. Loads & compiles the template, splits entries into Commands vs Events sections. |
| `template.html`  | Single embedded HTML/CSS template. Two-column layout (sidebar nav + main), collapsible `<details>` schemas, light/dark via `prefers-color-scheme`. |
| `util.go`        | `filepath.Abs` wrapper. |

## Constants worth knowing

- `loader.go:moduleRoot` = `"go.mau.fi/gomuks"` — the boundary between
  inline-expanded and link-only types.
- `loader.go:jsoncmdImportPath` = `moduleRoot + "/pkg/hicli/jsoncmd"` — the
  one package the generator is hardcoded to read.
- `schema.go:goBuiltins` — predeclared identifiers we render as basic types
  with no link.

## Common modifications

- **New spec type** (e.g. `CommandSpecBatched[...]`): add an entry to
  `specTypes` in `schema.go` and a corresponding `kind*` constant, then
  handle its type-argument layout in `extract.go:tryParseSpec`.
- **Different output sections**: edit the two `Section` constructions in
  `render.go:buildPage`.
- **Style / layout**: `template.html` only — everything is inline so the
  generated HTML stays single-file.
- **Different markdown flavor**: extend the goldmark instance in
  `render.go:renderMarkdown`.
- **Treat another package as "in-module"** (inline-expanded): change
  `moduleRoot` in `loader.go`, or generalize `isInModule`.
- **Surface a `TypeRef` field in the template**: also update
  `HasInlineStruct` if its presence should make the type collapsible.

## Known limits

- Only string-valued string-literal constants are indexed (`loader.go`
  `indexFile`). `Name:` fields pointing to typed-string constants resolve
  fine, but computed expressions don't.
- `parser.ParseDir` is used, so build tags / `//go:build` constraints are
  not evaluated — every non-`_test.go` file in the dir is read.
- External (out-of-module) types are never expanded inline, even if they're
  simple structs. They link to pkg.go.dev only. This is intentional
  (per `INITIAL_PROMPT.md`).
- The recursion guard is per-render-call, not global, so the same in-module
  type may be re-expanded across different commands. That's fine for output
  but means changes that explode schema size will show up as slower runs.
