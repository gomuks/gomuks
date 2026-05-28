// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

const (
	moduleRoot        = "go.mau.fi/gomuks"
	jsoncmdImportPath = moduleRoot + "/pkg/hicli/jsoncmd"
)

// pkg holds parsed AST info for a single Go package, indexed by name.
type pkg struct {
	importPath string
	dir        string
	files      []*ast.File
	fset       *token.FileSet

	// types is name → type spec (the *ast.TypeSpec) and the file it was declared in.
	types map[string]typeDecl
	// consts is name → string constant value (only constants with literal string values are stored).
	consts map[string]string
}

type typeDecl struct {
	spec *ast.TypeSpec
	file *ast.File
	doc  *ast.CommentGroup
}

type generator struct {
	root     string          // module root directory
	packages map[string]*pkg // import path → loaded package
	// fileImports maps a file's *ast.File pointer to local import alias -> full import path
	fileImports map[*ast.File]map[string]string
}

func newGenerator(root string) (*generator, error) {
	return &generator{
		root:        root,
		packages:    make(map[string]*pkg),
		fileImports: make(map[*ast.File]map[string]string),
	}, nil
}

// loadPackage loads a single Go package by import path. It only handles paths
// under the module root; external paths are ignored (they're rendered as
// pkg.go.dev links rather than expanded inline).
func (g *generator) loadPackage(importPath string) error {
	if _, ok := g.packages[importPath]; ok {
		return nil
	}
	if !g.isInModule(importPath) {
		return fmt.Errorf("not an in-module package: %s", importPath)
	}
	rel := strings.TrimPrefix(importPath, moduleRoot+"/")
	dir := filepath.Join(g.root, rel)

	fset := token.NewFileSet()
	parsed, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", dir, err)
	}

	// There may be multiple packages in a directory (e.g. xxx and xxx_test).
	// Pick the one whose name matches the basename (or the only one if just one).
	var chosen *ast.Package
	for _, p := range parsed {
		if !strings.HasSuffix(p.Name, "_test") {
			chosen = p
			break
		}
	}
	if chosen == nil {
		return fmt.Errorf("no package found in %s", dir)
	}

	loaded := &pkg{
		importPath: importPath,
		dir:        dir,
		fset:       fset,
		types:      make(map[string]typeDecl),
		consts:     make(map[string]string),
	}
	for _, f := range chosen.Files {
		loaded.files = append(loaded.files, f)
		g.indexFile(loaded, f)
	}
	g.packages[importPath] = loaded
	return nil
}

func (g *generator) indexFile(p *pkg, f *ast.File) {
	imports := make(map[string]string, len(f.Imports))
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := defaultImportAlias(path)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imports[name] = path
	}
	g.fileImports[f] = imports

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		switch gd.Tok {
		case token.TYPE:
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				doc := ts.Doc
				if doc == nil && len(gd.Specs) == 1 {
					doc = gd.Doc
				}
				p.types[ts.Name.Name] = typeDecl{spec: ts, file: f, doc: doc}
			}
		case token.CONST:
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						p.consts[name.Name] = strings.Trim(lit.Value, `"`)
					}
				}
			}
		}
	}
}

func (g *generator) isInModule(importPath string) bool {
	return importPath == moduleRoot || strings.HasPrefix(importPath, moduleRoot+"/")
}

// defaultImportAlias returns the conventional package name used for an import
// path when no alias is specified — i.e. the last path segment, after stripping
// a leading "vN" segment.
func defaultImportAlias(path string) string {
	last := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		last = path[idx+1:]
	}
	return last
}
