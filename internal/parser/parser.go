package parser

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"log"
	"path/filepath"
)

type ObjectCollection struct {
	structs    map[string]*types.Struct
	interfaces map[string]*types.Interface
}

func NewObjectCollections() *ObjectCollection {
	return &ObjectCollection{
		structs:    map[string]*types.Struct{},
		interfaces: map[string]*types.Interface{},
	}
}

func (oc *ObjectCollection) GetCollections(dir string) {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		fmt.Printf("walk path %s\n", path)
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			panic(err)
		}

		info := types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Defs:  make(map[*ast.Ident]types.Object),
			Uses:  make(map[*ast.Ident]types.Object),
		}

		for name, pkg := range pkgs {
			var files []*ast.File
			for _, f := range pkg.Files {
				files = append(files, f)
			}
			conf := types.Config{Importer: importer.Default()}
			fmt.Printf("check int name: %s\n", name)
			_, err = conf.Check(name, fset, files, &info)
			if err != nil {
				log.Fatal(err)
			}
			ast.Inspect(pkg, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.Ident:
					t := info.TypeOf(x)
					if t == nil {
						return true
					}
					if v, ok := t.Underlying().(*types.Struct); ok {
						fmt.Printf("%s is struct.\n", x.Name)
						oc.structs[x.Name] = v
					} else if v, ok := t.Underlying().(*types.Interface); ok {
						fmt.Printf("%s is interface.\n", x.Name)
						oc.interfaces[x.Name] = v
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func (oc *ObjectCollection) Dump() {
	fmt.Println(oc.structs)
	fmt.Println(oc.interfaces)
}
