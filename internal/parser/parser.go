package parser

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
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

func NewTypeGraph() *TypeGraph {
	return &TypeGraph{
		structs:    map[string]types.Object{},
		interfaces: map[string]types.Object{},
		edges:      []*edge{},
	}
}

func (tg *TypeGraph) handleExpr(expr ast.Expr, info *types.Info) []string {
	ret := []string{}

	fmt.Printf("expr: %s\n", types.ExprString(expr))

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
		ret = append(ret, tg.handleExpr(v.X, info)...)
	case *ast.ArrayType:
		ret = append(ret, tg.handleExpr(v.Elt, info)...)
	case *ast.MapType:
		ret = append(ret, tg.handleExpr(v.Key, info)...)
		ret = append(ret, tg.handleExpr(v.Value, info)...)
	case *ast.SelectorExpr:
		ret = append(ret, types.ExprString(v))
		// default:
		// 	fmt.Printf("type: %T\n", expr)
	}
	return ret
}

func (tg *TypeGraph) Build(path string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps,
		Dir: path,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}
	packages.PrintErrors(pkgs)

	for _, pkg := range pkgs {
		fmt.Printf("pkg: %s\n", pkg.Name)
		for _, syntax := range pkg.Syntax {
			fmt.Printf("file: %s\n", syntax.Name.Name)
			var parent types.Object
			ast.Inspect(syntax, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.TypeSpec:
					obj := pkg.TypesInfo.ObjectOf(x.Name)
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
					// TODO: x.TypeがStructTypeになっていて、そこから情報が取れそう。
				case *ast.StructType:
					for _, field := range x.Fields.List {
						// TODO: ST2 -> ST3 という組が2つできてしまう。
						// Map じゃないが、どうにか重複排除したい。
						strOrInterfaceNames := tg.handleExpr(field.Type, pkg.TypesInfo)
						for _, name := range strOrInterfaceNames {
							tg.edges = append(tg.edges, &edge{
								from: parent.Name(),
								to:   name,
								kind: Has,
							})
						}
					}
				case *ast.InterfaceType:
					for _, field := range x.Methods.List {
						// TODO: ST2 -> ST3 という組が2つできてしまう。
						// Map じゃないが、どうにか重複排除したい。
						strOrInterfaceNames := tg.handleExpr(field.Type, pkg.TypesInfo)
						for _, name := range strOrInterfaceNames {
							tg.edges = append(tg.edges, &edge{
								from: parent.Name(),
								to:   name,
								kind: Has,
							})
						}
					}
				}
				return true
			})
		}
	}
	return nil
}

func (tg *TypeGraph) Dump() {
	fmt.Println(tg.structs)
	fmt.Println(tg.interfaces)
	for _, edge := range tg.edges {
		fmt.Println(*edge)
	}
}
