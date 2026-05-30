package main

import (
	"go/parser"
	"testing"
)

func TestRenderTypeByteSliceUsesJSONShape(t *testing.T) {
	expr, err := parser.ParseExpr("[]byte")
	if err != nil {
		t.Fatal(err)
	}
	g := newGenerator(".")
	ref := g.renderType(&pkg{importPath: jsoncmdImportPath}, nil, expr, map[string]bool{})
	if ref.Kind != typeKindBasic {
		t.Fatalf("[]byte kind = %q, want %q", ref.Kind, typeKindBasic)
	}
	if ref.Display != "base64 string" {
		t.Fatalf("[]byte display = %q, want %q", ref.Display, "base64 string")
	}
}

func TestFlattenedFieldsHandlesMissingElement(t *testing.T) {
	for _, ref := range []*TypeRef{
		{Kind: typeKindSlice},
		{Kind: typeKindArray},
		{Kind: typeKindMap},
	} {
		if fields := ref.FlattenedFields(); fields != nil {
			t.Fatalf("FlattenedFields() = %#v, want nil", fields)
		}
	}
}
