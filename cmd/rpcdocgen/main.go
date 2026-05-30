// Command docgen generates an HTML reference for the JSON commands and events
// defined in the jsoncmd package.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type runConfig struct {
	outputPath string
	root       string
	stderr     io.Writer
}

func main() {
	output := flag.String("o", "jsoncmd.html", "Output HTML file path")
	root := flag.String("root", ".", "Module root directory (defaults to working directory)")
	flag.Parse()

	if err := run(runConfig{
		outputPath: *output,
		root:       *root,
		stderr:     os.Stderr,
	}); err != nil {
		log.Fatal(err)
	}
}

func run(cfg runConfig) error {
	if cfg.stderr == nil {
		cfg.stderr = io.Discard
	}

	absRoot, err := filepath.Abs(cfg.root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}

	g := newGenerator(absRoot)

	if err := g.loadPackage(jsoncmdImportPath); err != nil {
		return fmt.Errorf("load jsoncmd: %w", err)
	}

	specs, err := g.extractSpecs()
	if err != nil {
		return fmt.Errorf("extract specs: %w", err)
	}

	page, err := g.buildPage(specs)
	if err != nil {
		return fmt.Errorf("build page: %w", err)
	}

	if err := writePage(cfg.outputPath, page); err != nil {
		return err
	}
	fmt.Fprintf(cfg.stderr, "Wrote %d entries to %s\n", page.EntryCount(), cfg.outputPath)
	return nil
}

func writePage(outputPath string, page *Page) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if err := pageTemplate.Execute(f, page); err != nil {
		_ = f.Close()
		return fmt.Errorf("render template: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	return nil
}
