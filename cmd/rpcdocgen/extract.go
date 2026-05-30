package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
)

// specKind describes how a spec variable maps to request/response types.
type specKind int

const (
	kindCommand specKind = iota
	kindCommandNoResponse
	kindCommandNoRequest
	kindCommandNoData
	kindEvent
)

func (k specKind) isEvent() bool { return k == kindEvent }

// specTypes maps the bare type name in jsoncmd to its kind. The struct
// definitions themselves live in jsoncmd/spec.go.
var specTypes = map[string]specKind{
	"CommandSpec":                kindCommand,
	"CommandSpecWithoutResponse": kindCommandNoResponse,
	"CommandSpecWithoutRequest":  kindCommandNoRequest,
	"CommandSpecWithoutData":     kindCommandNoData,
	"EventSpec":                  kindEvent,
}

// rawSpec is a spec variable as extracted from the AST, before type schemas
// are resolved.
type rawSpec struct {
	varName    string
	cmdName    string // resolved string value of the Name constant
	doc        *ast.CommentGroup
	kind       specKind
	reqType    ast.Expr // nil if none
	respType   ast.Expr // nil if none
	file       *ast.File
	declarePos token.Pos
}

func (g *generator) extractSpecs() ([]*rawSpec, error) {
	p := g.packages[jsoncmdImportPath]
	if p == nil {
		return nil, fmt.Errorf("jsoncmd package not loaded")
	}

	var specs []*rawSpec
	for _, file := range p.files {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				doc := vs.Doc
				if doc == nil && len(gd.Specs) == 1 {
					doc = gd.Doc
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					rs := g.tryParseSpec(p, file, name.Name, vs.Values[i], doc)
					if rs != nil {
						rs.file = file
						rs.declarePos = name.Pos()
						specs = append(specs, rs)
					}
				}
			}
		}
	}

	sort.SliceStable(specs, func(i, j int) bool {
		return specs[i].declarePos < specs[j].declarePos
	})
	return specs, nil
}

// tryParseSpec inspects a single var value and returns a rawSpec if it
// matches one of the spec types, or nil otherwise.
func (g *generator) tryParseSpec(p *pkg, file *ast.File, varName string, value ast.Expr, doc *ast.CommentGroup) *rawSpec {
	// Unwrap pointer (e.g. &CommandSpec[...]{...}).
	if u, ok := value.(*ast.UnaryExpr); ok && u.Op == token.AND {
		value = u.X
	}
	cl, ok := value.(*ast.CompositeLit)
	if !ok || cl.Type == nil {
		return nil
	}

	typeIdent, typeArgs := splitGenericType(cl.Type)
	if typeIdent == nil {
		return nil
	}
	kind, ok := specTypes[typeIdent.Name]
	if !ok {
		return nil
	}

	rs := &rawSpec{
		varName: varName,
		doc:     doc,
		kind:    kind,
	}
	switch kind {
	case kindCommand:
		if len(typeArgs) >= 2 {
			rs.reqType = typeArgs[0]
			rs.respType = typeArgs[1]
		}
	case kindCommandNoResponse:
		if len(typeArgs) >= 1 {
			rs.reqType = typeArgs[0]
		}
	case kindCommandNoRequest:
		if len(typeArgs) >= 1 {
			rs.respType = typeArgs[0]
		}
	case kindCommandNoData:
		// no type args
	case kindEvent:
		if len(typeArgs) >= 1 {
			rs.respType = typeArgs[0]
		}
	}

	rs.cmdName = g.resolveNameField(p, cl)
	if rs.cmdName == "" {
		// Probably not a real spec, skip rather than emit incomplete docs.
		return nil
	}
	return rs
}

// splitGenericType peels apart a (possibly generic) type expression like
// CommandSpec[*Foo, *Bar] into (CommandSpec, [*Foo, *Bar]). Returns nil if the
// expression isn't an identifier or an indexed identifier.
func splitGenericType(expr ast.Expr) (*ast.Ident, []ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t, nil
	case *ast.IndexExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id, []ast.Expr{t.Index}
		}
	case *ast.IndexListExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id, t.Indices
		}
	}
	return nil, nil
}

// resolveNameField extracts the value of the Name field from a spec composite
// literal, then looks it up in the jsoncmd const table. Returns "" if not
// resolvable.
func (g *generator) resolveNameField(p *pkg, cl *ast.CompositeLit) string {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Name" {
			continue
		}
		switch v := kv.Value.(type) {
		case *ast.Ident:
			if val, ok := p.consts[v.Name]; ok {
				return val
			}
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				return trimQuotes(v.Value)
			}
		}
	}
	return ""
}

func trimQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '`') {
		return s[1 : len(s)-1]
	}
	return s
}
