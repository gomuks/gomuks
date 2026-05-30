// Command docgen generates an HTML reference for the JSON commands and events
// defined in the jsoncmd package.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	output := flag.String("o", "jsoncmd.html", "Output HTML file path")
	root := flag.String("root", ".", "Module root directory (defaults to working directory)")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
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

	page, err := g.buildPage(specs)
	if err != nil {
		log.Fatalf("build page: %v", err)
	}

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
