package main

import (
	"fmt"
	"os"
	"path/filepath"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"go.mau.fi/util/exerrors"
)

var CodeBlockFormatter = chromahtml.New(
	chromahtml.WithClasses(true),
	chromahtml.WithLineNumbers(true),
)

func main() {
	for _, name := range styles.Names() {
		path := filepath.Join(os.Args[len(os.Args)-1], fmt.Sprintf("%s.css", name))
		file := exerrors.Must(os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644))
		exerrors.PanicIfNotNil(CodeBlockFormatter.WriteCSS(file, styles.Get(name)))
	}
}
