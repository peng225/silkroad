package main

import (
	"github.com/peng225/silkroad/internal/parser"
)

func main() {
	tg := parser.NewTypeGraph()
	tg.Build("testdata/test1/test11")
	tg.Dump()
}
