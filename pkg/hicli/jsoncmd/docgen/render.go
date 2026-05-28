// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
)

// Page is the top-level data fed to the HTML template.
type Page struct {
	Title    string
	Sections []*Section
}

// Section groups entries (client→server commands vs server→client events).
type Section struct {
	ID      string
	Title   string
	Intro   template.HTML
	Entries []*Entry
}

// Entry is one CommandSpec or EventSpec rendered to the page.
type Entry struct {
	CmdName     string // the on-wire string name, e.g. "get_state"
	Doc         template.HTML
	Request     *TypeRef // may be nil
	Response    *TypeRef // may be nil
	HasRequest  bool
	HasResponse bool
	IsEvent     bool
	Anchor      string
}

// buildPage transforms raw extracted specs into the template-friendly Page
// structure, resolving request/response type schemas as it goes.
func (g *generator) buildPage(specs []*rawSpec) *Page {
	commands := &Section{
		ID:    "commands",
		Title: "Commands (client → server)",
		Intro: template.HTML("Requests the client can send to the server. Each command lists the request body and the response shape."),
	}
	events := &Section{
		ID:    "events",
		Title: "Events (server → client)",
		Intro: template.HTML("Asynchronous payloads pushed from the server to the client."),
	}

	jsoncmd := g.packages[jsoncmdImportPath]
	for _, rs := range specs {
		entry := &Entry{
			CmdName: rs.cmdName,
			IsEvent: rs.kind.isEvent(),
			Anchor:  anchorFor(rs.cmdName),
		}

		entry.Doc = renderEntryDoc(rs)

		if rs.reqType != nil {
			visited := map[string]bool{}
			entry.Request = g.renderType(jsoncmd, rs.file, rs.reqType, visited)
			entry.HasRequest = true
		}
		if rs.respType != nil {
			visited := map[string]bool{}
			entry.Response = g.renderType(jsoncmd, rs.file, rs.respType, visited)
			entry.HasResponse = true
		}

		if entry.IsEvent {
			events.Entries = append(events.Entries, entry)
		} else {
			commands.Entries = append(commands.Entries, entry)
		}
	}

	return &Page{
		Title:    "gomuks JSON command reference",
		Sections: []*Section{commands, events},
	}
}

// renderEntryDoc takes the doc comment from a spec variable and converts it
// to HTML, replacing the variable name at the start with the on-wire command
// name (e.g. "GetState returns..." → "`get_state` returns...").
func renderEntryDoc(rs *rawSpec) template.HTML {
	raw := commentText(rs.doc)
	raw = replaceLeadingVarName(raw, rs.varName, rs.cmdName)
	return renderMarkdown(raw)
}

// replaceLeadingVarName swaps the first occurrence of varName at the start of
// the doc string (optionally after whitespace) with a backticked cmdName,
// matching the Go doc-comment convention.
func replaceLeadingVarName(text, varName, cmdName string) string {
	trimmed := strings.TrimLeft(text, " \t\n")
	leadingWS := text[:len(text)-len(trimmed)]
	if !strings.HasPrefix(trimmed, varName) {
		return text
	}
	rest := trimmed[len(varName):]
	// Only treat it as a leading reference if what follows is whitespace or
	// punctuation — otherwise it might be part of a longer identifier.
	if rest != "" {
		c := rest[0]
		if !(c == ' ' || c == '\t' || c == '\n' || c == '.' || c == ',' || c == ':' || c == ';') {
			return text
		}
	}
	return leadingWS + "`" + cmdName + "`" + rest
}

func anchorFor(name string) string {
	return "cmd-" + strings.ReplaceAll(name, "_", "-")
}

// renderMarkdown converts a Markdown source string to safe HTML using goldmark.
// On error we fall back to escaped plain text so the page still renders.
func renderMarkdown(src string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(src), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	return template.HTML(buf.String())
}

//go:embed template.html
var pageTemplateSource string

var pageTemplate = template.Must(
	template.New("page").
		Funcs(template.FuncMap{
			"hasInline": func(t *TypeRef) bool { return t.HasInlineStruct() },
			"flattenedFields": func(t *TypeRef) []*Field {
				return t.FlattenedFields()
			},
			"flattenedFieldUnit": func(t *TypeRef) string {
				return t.FlattenedFieldUnit()
			},
		}).
		Parse(pageTemplateSource),
)
