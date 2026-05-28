// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Command docgen generates an HTML reference for the JSON commands and events
// defined in the jsoncmd package.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	output := flag.String("o", "jsoncmd.html", "Output HTML file path")
	root := flag.String("root", ".", "Module root directory (defaults to working directory)")
	flag.Parse()

	absRoot, err := absPath(*root)
	if err != nil {
		log.Fatalf("resolve root: %v", err)
	}

	g, err := newGenerator(absRoot)
	if err != nil {
		log.Fatalf("init generator: %v", err)
	}

	if err := g.loadPackage(jsoncmdImportPath); err != nil {
		log.Fatalf("load jsoncmd: %v", err)
	}

	specs, err := g.extractSpecs()
	if err != nil {
		log.Fatalf("extract specs: %v", err)
	}

	page := g.buildPage(specs)

	f, err := os.Create(*output)
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer f.Close()
	if err := pageTemplate.Execute(f, page); err != nil {
		log.Fatalf("render template: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Wrote %d entries to %s\n", len(page.Sections[0].Entries)+len(page.Sections[1].Entries), *output)
}
