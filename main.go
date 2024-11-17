package main

import (
	"github.com/peng225/silkroad/internal/parser"
)

func main() {
	oc := parser.NewObjectCollections()
	oc.GetCollections("testdata")
	oc.Dump()
}
