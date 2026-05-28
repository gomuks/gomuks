// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"go/ast"
	"html/template"
	"reflect"
	"strings"
)

// TypeRef is the rendered shape of a Go type, ready for the HTML template.
type TypeRef struct {
	Kind        string // basic, named, slice, array, map, struct, interface, func, chan, ellipsis, unknown
	Name        string
	PkgAlias    string // empty when in jsoncmd itself
	ImportPath  string
	Link        string // pkg.go.dev URL, or empty for basic types
	InModule    bool   // true if Link points inside this module
	Elem        *TypeRef
	Key         *TypeRef
	ArrayLen    string
	TypeArgs    []*TypeRef // for generic instances
	Fields      []*Field   // inline-expanded struct fields, when applicable
	EmptyStruct bool       // true when a named struct resolved to an empty object
	Recursive   bool       // we hit a cycle and stopped expanding here
	Underlying  string     // underlying JSON-ish type for named aliases, e.g. string
	Display     string     // fallback text for kinds we don't structure further
}

// Field is one struct field as it appears in the rendered schema.
type Field struct {
	JSONName string // computed from the json tag, or the Go name when no tag is set
	GoName   string
	Type     *TypeRef
	Doc      template.HTML // markdown rendered to HTML
	Optional bool          // tag has omitempty/omitzero
	Embedded bool          // Go-embedded field (no name)
	Skipped  bool          // tag is `json:"-"` — usually filtered out
}

// HasInlineStruct reports whether this TypeRef (or one of its element types,
// transitively) has expandable struct content for the template's <details> UI.
func (t *TypeRef) HasInlineStruct() bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case "struct":
		return len(t.Fields) > 0
	case "named":
		return len(t.Fields) > 0
	case "slice", "array":
		return t.Elem != nil && t.Elem.HasInlineStruct()
	case "map":
		return t.Elem != nil && t.Elem.HasInlineStruct()
	}
	return false
}

// DirectFields returns fields directly attached to this type, without walking
// through containers. The template uses this to flatten arrays of structs.
func (t *TypeRef) DirectFields() []*Field {
	if t == nil {
		return nil
	}
	switch t.Kind {
	case "struct", "named":
		return t.Fields
	}
	return nil
}

// FlattenedFields returns fields that can be shown directly when a collection
// type is opened. It unwraps containers until it reaches a struct-like value.
func (t *TypeRef) FlattenedFields() []*Field {
	if t == nil {
		return nil
	}
	if fields := t.DirectFields(); len(fields) > 0 {
		return fields
	}
	switch t.Kind {
	case "slice", "array", "map":
		return t.Elem.FlattenedFields()
	}
	return nil
}

func (t *TypeRef) FlattenedFieldUnit() string {
	if t == nil {
		return "value"
	}
	switch t.Kind {
	case "slice", "array":
		return "item"
	case "map":
		if t.Elem != nil && (t.Elem.Kind == "slice" || t.Elem.Kind == "array") {
			return "value item"
		}
		return "value"
	}
	return "value"
}

// goBuiltins are the predeclared identifiers we treat as basic types (no link).
var goBuiltins = map[string]bool{
	"string": true, "bool": true, "byte": true, "rune": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true,
	"complex64": true, "complex128": true,
	"error": true, "any": true,
}

var noInlineTypes = map[string]bool{
	"maunium.net/go/mautrix/event.MessageEventContent": true,
	"maunium.net/go/mautrix/event.Unsigned":            true,
}

var customJSONTypeUnderlying = map[string]string{
	"encoding/json.RawMessage": "arbitrary JSON",

	"maunium.net/go/mautrix/event.Type":    "string",
	"maunium.net/go/mautrix/event.Content": "arbitrary JSON object",
	"maunium.net/go/mautrix/id.ContentURI": "string",
	"maunium.net/go/mautrix/id.TrustState": "string",

	"maunium.net/go/mautrix.Direction": "string, 1 character",

	"go.mau.fi/util/jsontime.Unix":      "int64",
	"go.mau.fi/util/jsontime.UnixMilli": "int64",
	"go.mau.fi/util/jsontime.UnixMicro": "int64",
	"go.mau.fi/util/jsontime.UnixNano":  "int64",

	"go.mau.fi/util/jsontime.Seconds":      "int64",
	"go.mau.fi/util/jsontime.Milliseconds": "int64",
	"go.mau.fi/util/jsontime.Microseconds": "int64",
	"go.mau.fi/util/jsontime.Nanoseconds":  "int64",

	"go.mau.fi/util/jsontime.UnixString":      "int64 in string",
	"go.mau.fi/util/jsontime.UnixMilliString": "int64 in string",
	"go.mau.fi/util/jsontime.UnixMicroString": "int64 in string",
	"go.mau.fi/util/jsontime.UnixNanoString":  "int64 in string",
}

// renderType turns an AST type expression into a TypeRef tree. file gives us
// the import scope; visited tracks which named types are currently being
// expanded so we don't recurse forever on cyclic schemas.
func (g *generator) renderType(p *pkg, file *ast.File, expr ast.Expr, visited map[string]bool) *TypeRef {
	if expr == nil {
		return &TypeRef{Kind: "unknown", Display: "?"}
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return g.renderIdent(p, t, visited)
	case *ast.SelectorExpr:
		return g.renderSelector(p, file, t, visited)
	case *ast.StarExpr:
		// JSON-wise, a pointer is the same as the underlying value (with
		// nullability), so we just unwrap it.
		return g.renderType(p, file, t.X, visited)
	case *ast.ArrayType:
		ref := &TypeRef{Kind: "slice", Elem: g.renderType(p, file, t.Elt, visited)}
		if t.Len != nil {
			ref.Kind = "array"
			ref.ArrayLen = exprString(t.Len)
		}
		// Special case: []byte is usually base64-encoded JSON strings.
		if id, ok := t.Elt.(*ast.Ident); ok && id.Name == "byte" && t.Len == nil {
			return &TypeRef{Kind: "basic", Name: "[]byte", Display: "[]byte"}
		}
		return ref
	case *ast.MapType:
		return &TypeRef{
			Kind: "map",
			Key:  g.renderType(p, file, t.Key, visited),
			Elem: g.renderType(p, file, t.Value, visited),
		}
	case *ast.StructType:
		return g.renderStructFields(p, file, t, visited)
	case *ast.InterfaceType:
		return &TypeRef{Kind: "interface", Display: "any"}
	case *ast.IndexExpr:
		return g.renderGenericInstance(p, file, t.X, []ast.Expr{t.Index}, visited)
	case *ast.IndexListExpr:
		return g.renderGenericInstance(p, file, t.X, t.Indices, visited)
	case *ast.FuncType:
		return &TypeRef{Kind: "func", Display: "function"}
	case *ast.ChanType:
		return &TypeRef{Kind: "chan", Display: "channel"}
	case *ast.Ellipsis:
		ref := &TypeRef{Kind: "slice", Elem: g.renderType(p, file, t.Elt, visited)}
		return ref
	}
	return &TypeRef{Kind: "unknown", Display: fmt.Sprintf("<unknown %T>", expr)}
}

// renderIdent handles a bare identifier — either a Go builtin or a type
// declared in the current package.
func (g *generator) renderIdent(p *pkg, id *ast.Ident, visited map[string]bool) *TypeRef {
	if goBuiltins[id.Name] {
		return &TypeRef{Kind: "basic", Name: id.Name, Display: id.Name}
	}
	return g.renderNamedRef(p.importPath, id.Name, "", visited)
}

// renderSelector handles a qualified identifier like database.Event. The X
// must be a package alias visible in the file's import scope.
func (g *generator) renderSelector(p *pkg, file *ast.File, sel *ast.SelectorExpr, visited map[string]bool) *TypeRef {
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return &TypeRef{Kind: "unknown", Display: exprString(sel)}
	}
	imports := g.fileImports[file]
	importPath, ok := imports[pkgIdent.Name]
	if !ok {
		// Probably an enum constant reference or another package without an
		// explicit import alias — fall back to displaying the source text.
		return &TypeRef{Kind: "unknown", Display: exprString(sel)}
	}
	return g.renderNamedRef(importPath, sel.Sel.Name, pkgIdent.Name, visited)
}

// renderNamedRef looks up a named type by (importPath, typeName) and produces
// a TypeRef. Structs are expanded inline unless a specific type is blacklisted.
func (g *generator) renderNamedRef(importPath, typeName, alias string, visited map[string]bool) *TypeRef {
	if alias == "" && importPath != jsoncmdImportPath {
		alias = defaultImportAlias(importPath)
	}
	ref := &TypeRef{
		Kind:       "named",
		Name:       typeName,
		PkgAlias:   alias,
		ImportPath: importPath,
		Link:       pkgGoDevURL(importPath, typeName),
		InModule:   isInModule(importPath),
	}

	key := importPath + "." + typeName
	if underlying, ok := customJSONTypeUnderlying[key]; ok {
		ref.Underlying = underlying
		return ref
	}

	if err := g.loadPackage(importPath); err != nil {
		return ref
	}
	loaded := g.packages[importPath]
	if loaded == nil {
		return ref
	}
	td, ok := loaded.types[typeName]
	if !ok {
		return ref
	}

	if visited[key] {
		ref.Recursive = true
		return ref
	}
	nextVisited := cloneVisited(visited)
	nextVisited[key] = true

	st, ok := td.spec.Type.(*ast.StructType)
	if !ok {
		ref.Underlying = g.underlyingTypeText(loaded, td.file, td.spec.Type, nextVisited)
		return ref
	}
	if noInlineTypes[key] {
		return ref
	}

	inner := g.renderStructFields(loaded, td.file, st, nextVisited)
	ref.Fields = inner.Fields
	ref.EmptyStruct = len(inner.Fields) == 0
	return ref
}

// renderGenericInstance handles things like `Container[Foo, Bar]`. The base
// type X is rendered as a named ref and the type arguments are attached.
func (g *generator) renderGenericInstance(p *pkg, file *ast.File, base ast.Expr, args []ast.Expr, visited map[string]bool) *TypeRef {
	baseRef := g.renderType(p, file, base, visited)
	for _, a := range args {
		baseRef.TypeArgs = append(baseRef.TypeArgs, g.renderType(p, file, a, visited))
	}
	return baseRef
}

// renderStructFields turns a struct type body into a TypeRef of kind "struct",
// recursively rendering each field's type and pulling docs out of the AST.
func (g *generator) renderStructFields(p *pkg, file *ast.File, st *ast.StructType, visited map[string]bool) *TypeRef {
	ref := &TypeRef{Kind: "struct"}
	if st.Fields == nil {
		return ref
	}
	for _, field := range st.Fields.List {
		fieldTypeRef := g.renderType(p, file, field.Type, visited)

		tag := parseStructTag(field.Tag)
		jsonName, optional, skipped := parseJSONTag(tag.Get("json"))

		docHTML := combineDocs(field.Doc, field.Comment)

		if len(field.Names) == 0 {
			if skipped {
				continue
			}
			if jsonName == "" {
				if flattened := cloneFields(fieldTypeRef.DirectFields(), optional); len(flattened) > 0 {
					ref.Fields = append(ref.Fields, flattened...)
					continue
				}
			}
			// Embedded field. Use the type's terminal name as the display name.
			name := embeddedDisplayName(field.Type)
			if jsonName == "" && name == "" {
				continue
			}
			f := &Field{
				JSONName: jsonName,
				GoName:   name,
				Type:     fieldTypeRef,
				Doc:      docHTML,
				Optional: optional,
				Embedded: true,
				Skipped:  skipped,
			}
			if jsonName == "" {
				f.JSONName = name
			}
			ref.Fields = append(ref.Fields, f)
			continue
		}

		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			displayName := jsonName
			if displayName == "" {
				displayName = name.Name
			}
			f := &Field{
				JSONName: displayName,
				GoName:   name.Name,
				Type:     fieldTypeRef,
				Doc:      docHTML,
				Optional: optional,
				Skipped:  skipped,
			}
			if !skipped {
				ref.Fields = append(ref.Fields, f)
			}
		}
	}
	return ref
}

func parseStructTag(tag *ast.BasicLit) reflect.StructTag {
	if tag == nil {
		return ""
	}
	// tag.Value includes the surrounding backticks (or quotes).
	v := tag.Value
	if len(v) >= 2 {
		v = v[1 : len(v)-1]
	}
	return reflect.StructTag(v)
}

// parseJSONTag splits a json struct tag into (name, optional, skipped).
// If the tag is "-" the field is skipped. "omitempty"/"omitzero" mark optional.
func parseJSONTag(tag string) (name string, optional bool, skipped bool) {
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return "", false, false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, p := range parts[1:] {
		if p == "omitempty" || p == "omitzero" {
			optional = true
		}
	}
	return name, optional, false
}

// embeddedDisplayName returns the terminal type name of an embedded field
// expression, peeling off pointers and selectors.
func embeddedDisplayName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return embeddedDisplayName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.IndexExpr:
		return embeddedDisplayName(t.X)
	case *ast.IndexListExpr:
		return embeddedDisplayName(t.X)
	}
	return ""
}

func cloneVisited(v map[string]bool) map[string]bool {
	out := make(map[string]bool, len(v)+1)
	for k := range v {
		out[k] = true
	}
	return out
}

func cloneFields(fields []*Field, optional bool) []*Field {
	if len(fields) == 0 {
		return nil
	}
	out := make([]*Field, 0, len(fields))
	for _, field := range fields {
		if field == nil || field.Skipped {
			continue
		}
		cloned := *field
		cloned.Embedded = false
		cloned.Optional = cloned.Optional || optional
		out = append(out, &cloned)
	}
	return out
}

func (g *generator) underlyingTypeText(p *pkg, file *ast.File, expr ast.Expr, visited map[string]bool) string {
	ref := g.renderType(p, file, expr, visited)
	if ref.Kind == "named" && ref.Underlying != "" {
		return ref.Underlying
	}
	return plainTypeLabel(ref)
}

func plainTypeLabel(t *TypeRef) string {
	if t == nil {
		return "?"
	}
	switch t.Kind {
	case "basic":
		if t.Display != "" {
			return t.Display
		}
		return t.Name
	case "named":
		var b strings.Builder
		if t.PkgAlias != "" {
			b.WriteString(t.PkgAlias)
			b.WriteByte('.')
		}
		b.WriteString(t.Name)
		if len(t.TypeArgs) > 0 {
			b.WriteByte('[')
			for i, arg := range t.TypeArgs {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(plainTypeLabel(arg))
			}
			b.WriteByte(']')
		}
		return b.String()
	case "slice":
		return "[]" + plainTypeLabel(t.Elem)
	case "array":
		return "[" + t.ArrayLen + "]" + plainTypeLabel(t.Elem)
	case "map":
		return "map[" + plainTypeLabel(t.Key) + "]" + plainTypeLabel(t.Elem)
	case "struct":
		return "object"
	case "interface":
		return "any"
	case "func":
		return "function"
	case "chan":
		return "channel"
	}
	if t.Display != "" {
		return t.Display
	}
	return "?"
}

func isInModule(importPath string) bool {
	return importPath == moduleRoot || strings.HasPrefix(importPath, moduleRoot+"/")
}

func pkgGoDevURL(importPath, typeName string) string {
	return "https://pkg.go.dev/" + importPath + "#" + typeName
}

// exprString returns a best-effort source-like rendering of an expression for
// use in fallback display strings.
func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.ArrayType:
		if t.Len != nil {
			return "[" + exprString(t.Len) + "]" + exprString(t.Elt)
		}
		return "[]" + exprString(t.Elt)
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.BasicLit:
		return t.Value
	case *ast.IndexExpr:
		return exprString(t.X) + "[" + exprString(t.Index) + "]"
	case *ast.IndexListExpr:
		var b strings.Builder
		b.WriteString(exprString(t.X))
		b.WriteByte('[')
		for i, index := range t.Indices {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(exprString(index))
		}
		b.WriteByte(']')
		return b.String()
	}
	return fmt.Sprintf("<%T>", e)
}

// combineDocs merges leading doc and trailing line comments on a struct field
// into a single Markdown-rendered HTML blob.
func combineDocs(doc, comment *ast.CommentGroup) template.HTML {
	var b strings.Builder
	if doc != nil {
		b.WriteString(commentText(doc))
	}
	if comment != nil {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(commentText(comment))
	}
	if b.Len() == 0 {
		return ""
	}
	return renderMarkdown(b.String())
}

func commentText(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	var b strings.Builder
	for i, c := range g.List {
		if i > 0 {
			b.WriteByte('\n')
		}
		text := c.Text
		switch {
		case strings.HasPrefix(text, "//"):
			text = strings.TrimPrefix(text, "//")
			text = strings.TrimPrefix(text, " ")
		case strings.HasPrefix(text, "/*"):
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)
		}
		b.WriteString(text)
	}
	return b.String()
}
