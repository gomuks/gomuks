package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
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

type goListPackage struct {
	Dir      string
	GoFiles  []string
	CgoFiles []string
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

// loadPackage loads a single Go package by import path. It uses `go list` so
// external packages and stdlib packages can be parsed shallowly for type aliases
// and top-level request/response schemas.
func (g *generator) loadPackage(importPath string) error {
	if _, ok := g.packages[importPath]; ok {
		return nil
	}

	listed, err := g.goList(importPath)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	loaded := &pkg{
		importPath: importPath,
		dir:        listed.Dir,
		fset:       fset,
		types:      make(map[string]typeDecl),
		consts:     make(map[string]string),
	}
	goFiles := append([]string{}, listed.GoFiles...)
	goFiles = append(goFiles, listed.CgoFiles...)
	for _, name := range goFiles {
		path := filepath.Join(listed.Dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		loaded.files = append(loaded.files, f)
	}
	if len(loaded.files) == 0 {
		return fmt.Errorf("no Go files found in %s", listed.Dir)
	}
	for _, f := range loaded.files {
		g.indexFile(loaded, f)
	}
	g.packages[importPath] = loaded
	return nil
}

func (g *generator) goList(importPath string) (*goListPackage, error) {
	cmd := exec.Command("go", "list", "-json", importPath)
	cmd.Dir = g.root
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr == "" {
			return nil, fmt.Errorf("go list %s: %w", importPath, err)
		}
		return nil, fmt.Errorf("go list %s: %w: %s", importPath, err, stderr)
	}
	var listed goListPackage
	if err := json.Unmarshal(out, &listed); err != nil {
		return nil, fmt.Errorf("decode go list %s: %w", importPath, err)
	}
	if listed.Dir == "" {
		return nil, fmt.Errorf("go list %s returned no directory", importPath)
	}
	return &listed, nil
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
