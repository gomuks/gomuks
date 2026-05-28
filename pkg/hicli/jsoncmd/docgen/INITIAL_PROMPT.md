Design and implement a documentation generator in Go, which parses docstrings and types from Go code and emits a single HTML file with all of the docs.

The target to parse is in pkg/hicli/jsoncmd/commands.go: each CommandSpec variable is a command that clients can send, plus there are EventSpec variables for commands that the server can send.
The variable comment has the main documentation.
The type parameters contain the request and response types if applicable.
The `Name` constant inside the struct definition is the actual name of the command.

The documentation should consist of the actual command name, the main documentation, and the request and response type schemas (which can have their own docstrings inside the struct definition).
The schemas should be nicely rendered, probably collapsed by default so nested fields can be opened easily.
Types that are outside of the current module (`go.mau.fi/gomuks`), e.g. in mautrix-go, should linkify to pkg.go.dev rather than being rendered inline.
In-module types should *also* link to pkg.go.dev in addition to having an inline schema.
Have the generator replace the variable name at the start of the docstring with the actual command name.

Put the generator in ./pkg/hicli/jsoncmd/docgen/
Use Go's html/template. Use github.com/yuin/goldmark for parsing markdown in comments.
The generator doesn't need to support arbitrary input packages: it should be hardcoded to read the jsoncmd package.
The generator must support commands being added and removed, so names, body types, etc must NOT be hardcoded.
The `CommandSpec`* and `EventSpec` type references can be hardcoded.
