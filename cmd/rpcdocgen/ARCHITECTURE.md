# rpcdocgen - jsoncmd reference generator

`rpcdocgen` is a standalone Go program that parses `pkg/hicli/jsoncmd` and emits a
single self-contained HTML reference for the RPC API. It is intentionally
hardcoded to the jsoncmd package, but it discovers commands, events, names, body
types, and comments from the Go AST so command additions/removals normally do
not require generator changes.

The generated page contains:

- Top-level Markdown docs from `pkg/hicli/jsoncmd/envelope.md` and
  `pkg/hicli/jsoncmd/websocket.md`.
- A Commands section for frontend-to-backend requests.
- An Events section for backend-to-frontend payloads.
- A sidebar with top-level links for Envelope, Websocket, Commands, and Events,
  plus command/event anchors under their sections.

## Running

From the module root:

```sh
go run ./cmd/rpcdocgen -o jsoncmd.html
```

Flags:

- `-o` - output path, default `jsoncmd.html`.
- `-root` - module root directory, default `.`.

The package should pass:

```sh
go vet ./cmd/rpcdocgen
staticcheck ./cmd/rpcdocgen
```

staticcheck can be installed with `go install honnef.co/go/tools/cmd/staticcheck@latest` if it's not available

## Inputs

The generator reads only the jsoncmd package and its imported type packages. The
main inputs are:

- `commands.go` - package-level `var` specs for commands and events.
- `envelope.md` - rendered before command/event docs.
- `websocket.md` - rendered after Envelope and before command/event docs.
- Struct/type declarations in jsoncmd, in-module packages, external modules, and
  stdlib packages as needed for schemas.

Markdown is converted with `github.com/yuin/goldmark`. Go AST parsing uses
`go/parser`, with package directories resolved through `go list -json`.

## Spec Extraction

`extract.go` walks package-level `var` declarations and recognizes composite
literals whose type is one of these hardcoded spec types:

| Type                              | Generic shape      | Section  |
| --------------------------------- | ------------------ | -------- |
| `CommandSpec[Req, Resp]`          | request + response | Commands |
| `CommandSpecWithoutResponse[Req]` | request only       | Commands |
| `CommandSpecWithoutRequest[Resp]` | response only      | Commands |
| `CommandSpecWithoutData`          | neither            | Commands |
| `EventSpec[Payload]`              | payload only       | Events   |

For each match, it extracts:

1. The on-wire name from the composite literal's `Name:` field. This may be a
   string literal or an identifier that resolves to a string constant in the
   jsoncmd package.
2. The var doc comment as the entry description. `render.go` replaces a leading
   Go variable name in that comment with the actual on-wire command/event name.
3. Request/response/payload type expressions from generic type arguments.

Entries without a resolvable `Name` are skipped. Extracted entries are sorted by
source position, so the HTML follows the source order in `commands.go`.

Go variable names are not rendered in the HTML.

## Package Loading

`loader.go` resolves packages with:

```sh
go list -json <import path>
```

It then parses the package's `GoFiles` and `CgoFiles`, indexing:

- Type declarations by name.
- String literal constants by name, used for resolving `Name:` fields.
- Per-file import aliases, used to resolve selectors such as `database.Event`.

Packages are cached by import path. External packages and stdlib packages are
parsed too, because schemas can inline external structs unless explicitly
blocked.

## Schema Rendering

`schema.go` converts `go/ast` type expressions into a `TypeRef` tree consumed by
`template.html`.

Important behavior:

- Pointers are unwrapped, because JSON shape is the underlying value plus
  possible nullability.
- Builtins render as basic unlinked types.
- `[]byte` renders as a base64 string instead of an expandable byte slice.
- Named types link to pkg.go.dev. Package-qualified names such as `id.RoomID`
  and `database.Event` are fully included in the link text.
- Type aliases display their underlying JSON-ish type, e.g. `id.RoomID
  (string)`.
- `json.RawMessage` displays as `arbitrary JSON`.
- Named empty structs display as `(empty object)` instead of a bare unopenable
  type.
- Recursive type expansion is stopped with a `visited` map keyed by
  `importPath.TypeName`; recursive references are marked in the HTML.
- Generic instances preserve and render their type arguments.
- Interfaces render as `any`; functions and channels render as fallback labels.

### Struct Fields

Struct fields are rendered from their JSON shape:

- `json` tags decide field names.
- `omitempty` and `omitzero` add an `optional` tag.
- `json:"-"` fields are skipped.
- Unexported named fields are skipped.
- Field doc comments are rendered through Markdown.
- Anonymous embedded structs with no explicit JSON name are flattened into the
  parent schema when their fields are known. This makes embedded types such as
  `mautrix.PublicRoomInfo`, `ReqSendToDevice`, `RecoveryKeyResponse`, and
  `event.TypingEventContent` appear as normal parent fields.
- Anonymous embedded fields that cannot be flattened are still shown as embedded
  fields using their terminal type name.

### External Types

External structs are inlined by default. The explicit blacklist
`schema.go:noInlineTypes` keeps types that are too broad or misleading as linked
leaves. At the time of writing, this includes:

- `maunium.net/go/mautrix/event.MessageEventContent`
- `maunium.net/go/mautrix/event.Unsigned`

Types with custom JSON marshaling that cannot be inferred reliably from their
struct fields are handled by `schema.go:customJSONTypeUnderlying`. Examples
include:

- `encoding/json.RawMessage` -> arbitrary JSON.
- `event.Type` -> string.
- `event.Content` -> arbitrary JSON object.
- `id.ContentURI` -> string.
- `id.TrustState` -> string.
- `mautrix.Direction` -> one-character string.
- `go.mau.fi/util/jsontime` Unix and duration wrappers -> integer shapes, and
  string wrappers -> integer-in-string shapes.

Add mappings there when a type's Go fields do not match its JSON encoding.

### Collection Flattening

Slices, arrays, and maps with struct-like values are flattened in the UI. Opening
the collection's `<details>` shows the value/item fields directly instead of
requiring a second click to open the element schema. For maps of slices, the
label says fields are per value item.

## Rendering

`render.go` turns extracted specs into a `Page`:

- It reads and renders the two top-level Markdown docs.
- It splits entries into Commands and Events.
- It resolves request/response/payload schemas.
- It feeds the embedded `template.html` through Go's `html/template`.

The template is a single-file responsive HTML page:

- Two-column layout on desktop.
- Sidebar hidden on narrow screens.
- Light/dark colors via `prefers-color-scheme`.
- Schemas use collapsible `<details>` blocks.
- Leaf types use compact code-like labels.

## File Map

| File            | Responsibility |
| --------------- | -------------- |
| `main.go`       | CLI entry point, loads jsoncmd, extracts specs, builds the page, executes the template. |
| `loader.go`     | Resolves package directories with `go list`, parses Go files, indexes types, constants, and imports. |
| `extract.go`    | Finds command/event spec variables, resolves on-wire names, extracts type arguments. |
| `schema.go`     | Converts AST type expressions to rendered schema trees, expands fields, handles links, aliases, recursion, custom JSON types, and flattening. |
| `render.go`     | Builds `Page` data, renders Markdown docs/comments, wires template helpers. |
| `template.html` | Embedded HTML/CSS/template for the generated page. |

## Common Changes

- New command/event spec wrapper: add it to `extract.go:specTypes`, add a
  `kind*` value if needed, and handle its type argument layout in
  `tryParseSpec`.
- New top-level Markdown doc: add the source file to jsoncmd and extend
  `render.go:buildDocSections`; add sidebar/main rendering if the structure
  differs from existing docs.
- Type with custom JSON encoding: add a `customJSONTypeUnderlying` entry.
- Type that should stay linked only: add a `noInlineTypes` entry.
- Layout/style changes: edit `template.html`.
- Different Markdown behavior: configure the goldmark call in
  `render.go:renderMarkdown`.

## Known Limits

- Only string literal constants are indexed for resolving `Name:` fields.
  Computed string expressions are not resolved.
- The schema walker is AST-based, not type-checked. It handles the patterns used
  by jsoncmd and its dependencies, but it does not evaluate arbitrary Go.
- Custom JSON behavior is hardcoded when needed; it is not inferred from
  `MarshalJSON` methods.
- The recursion guard is per top-level render call, so the same type may be
  expanded separately in multiple command/event schemas.
