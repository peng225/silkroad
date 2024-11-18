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

// TODO: need package info
type TypeGraph struct {
	structs    map[string]types.Object
	interfaces map[string]types.Object
	edges      []*edge
}

type edgeKind int

const (
	Has edgeKind = iota
	Implements
	Embeds
)

type edge struct {
	from string
	to   string
	kind edgeKind
}

var info types.Info

func init() {
	info = types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
}

func NewTypeGraph() *TypeGraph {
	return &TypeGraph{
		structs:    map[string]types.Object{},
		interfaces: map[string]types.Object{},
		edges:      []*edge{},
	}
}

func (tg *TypeGraph) handleExpr(expr ast.Expr) []string {
	ret := []string{}

	t := info.TypeOf(expr)
	if t == nil {
		return nil
	}
	switch t.Underlying().(type) {
	case *types.Struct:
		ret = append(ret, types.ExprString(expr))
		return ret
	case *types.Interface:
		ret = append(ret, types.ExprString(expr))
		return ret

	}

	switch v := expr.(type) {
	case *ast.StarExpr:
		ret = append(ret, tg.handleExpr(v.X)...)
	case *ast.ArrayType:
		ret = append(ret, tg.handleExpr(v.Elt)...)
	case *ast.MapType:
		ret = append(ret, tg.handleExpr(v.Key)...)
		ret = append(ret, tg.handleExpr(v.Value)...)
	default:
	}
	return ret
}

func (tg *TypeGraph) Build(dir string) {
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

		for name, pkg := range pkgs {
			var files []*ast.File
			for _, f := range pkg.Files {
				files = append(files, f)
			}
			conf := types.Config{Importer: importer.Default()}
			fmt.Printf("check int name: %s\n", name)
			_, err = conf.Check(path, fset, files, &info)
			if err != nil {
				log.Fatal(err)
			}

			var parent types.Object
			ast.Inspect(pkg, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.StructType:
					for _, field := range x.Fields.List {
						// ST2 -> ST3 という組が2つできてしまう。
						// Map じゃないが、どうにか重複排除したい。
						strOrInterfaceNames := tg.handleExpr(field.Type)
						for _, name := range strOrInterfaceNames {
							tg.edges = append(tg.edges, &edge{
								from: parent.Name(),
								to:   name,
								kind: Has,
							})
						}
					}
				case *ast.TypeSpec:
					obj := info.ObjectOf(x.Name)
					if obj == nil {
						return true
					}
					if _, ok := obj.Type().Underlying().(*types.Struct); ok {
						fmt.Printf("%s is struct.\n", obj.Name())
						tg.structs[obj.Name()] = obj
						parent = obj
					} else if _, ok := obj.Type().Underlying().(*types.Interface); ok {
						fmt.Printf("%s is interface.\n", obj.Name())
						tg.interfaces[obj.Name()] = obj
						parent = obj
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

func (tg *TypeGraph) Dump() {
	fmt.Println(tg.structs)
	fmt.Println(tg.interfaces)
	for _, edge := range tg.edges {
		fmt.Println(*edge)
	}
}
