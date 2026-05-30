package main

import "testing"

func TestDefaultImportAlias(t *testing.T) {
	tests := map[string]string{
		"encoding/json":               "json",
		"github.com/gdamore/tcell/v2": "tcell",
		"mvdan.cc/xurls/v2":           "xurls",
		"gopkg.in/yaml.v3":            "yaml",
		"example.com/v2/subpkg":       "subpkg",
	}
	for path, want := range tests {
		if got := defaultImportAlias(path); got != want {
			t.Errorf("defaultImportAlias(%q) = %q, want %q", path, got, want)
		}
	}
}
