package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
)

// Page is the top-level data fed to the HTML template.
type Page struct {
	Title    string
	Docs     []*DocSection
	Sections []*Section
}

// EntryCount returns the total number of command and event entries across all
// sections.
func (p *Page) EntryCount() int {
	n := 0
	for _, s := range p.Sections {
		n += len(s.Entries)
	}
	return n
}

// DocSection is a standalone markdown document rendered before the command list.
type DocSection struct {
	ID    string
	Title string
	HTML  template.HTML
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
func (g *generator) buildPage(specs []*rawSpec) (*Page, error) {
	commands := &Section{
		ID:    "commands",
		Title: "Commands",
		Intro: template.HTML("Requests that frontends can send to the backend."),
	}
	events := &Section{
		ID:    "events",
		Title: "Events",
		Intro: template.HTML("Events that the backend will send to connected frontends."),
	}

	jsoncmd := g.packages[jsoncmdImportPath]
	docs, err := g.buildDocSections(jsoncmd)
	if err != nil {
		return nil, err
	}
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
		Title:    "Gomuks RPC API",
		Docs:     docs,
		Sections: []*Section{commands, events},
	}, nil
}

func (g *generator) buildDocSections(jsoncmd *pkg) ([]*DocSection, error) {
	if jsoncmd == nil {
		return nil, fmt.Errorf("jsoncmd package not loaded")
	}
	docs := []struct {
		id       string
		title    string
		filename string
	}{
		{id: "envelope", title: "Envelope", filename: "envelope.md"},
		{id: "websocket", title: "Websocket", filename: "websocket.md"},
	}
	out := make([]*DocSection, 0, len(docs))
	for _, doc := range docs {
		path := filepath.Join(jsoncmd.dir, doc.filename)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		out = append(out, &DocSection{
			ID:    doc.id,
			Title: doc.title,
			HTML:  renderMarkdown(string(raw)),
		})
	}
	return out, nil
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
