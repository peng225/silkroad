package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// TODO: need package info
type TypeGraph struct {
	structs    map[string](map[string]string)
	interfaces map[string](map[string]string)
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

type importInfo struct {
	alias string
	path  string
}

func NewTypeGraph() *TypeGraph {
	return &TypeGraph{
		structs:    map[string](map[string]string){},
		interfaces: map[string](map[string]string){},
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

func (tg *TypeGraph) buildEdge(fields []*ast.Field, info *types.Info, parent types.Object, ii []importInfo) {
	for _, field := range fields {
		// TODO: ST2 -> ST3 という組が2つできてしまう。
		// Map じゃないが、どうにか重複排除したい。
		strOrInterfaceNames := tg.handleExpr(field.Type, info)
		for _, name := range strOrInterfaceNames {
			if name == "struct{}" {
				continue
			}
			fullName := parent.Pkg().Path() + "." + name
			for _, v := range ii {
				tokens := strings.Split(name, ".")
				if len(tokens) == 2 && v.alias == tokens[0] {
					fullName = v.path + "." + tokens[1]
					break
				} else if len(tokens) == 2 && strings.HasSuffix(v.path, tokens[0]) {
					fullName = v.path + "." + tokens[1]
					break
				}
			}
			tg.edges = append(tg.edges, &edge{
				from: parent.Pkg().Path() + "." + parent.Name(),
				to:   fullName,
				kind: Has,
			})
		}
	}
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
			ii := []importInfo{}
			ast.Inspect(syntax, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ImportSpec:
					iiEntry := importInfo{
						path: strings.Trim(x.Path.Value, `"`),
					}
					if x.Name != nil {
						iiEntry.alias = x.Name.Name
					}
					ii = append(ii, iiEntry)
				case *ast.TypeSpec:
					obj := pkg.TypesInfo.ObjectOf(x.Name)
					if obj == nil {
						return true
					}
					if _, ok := obj.Type().Underlying().(*types.Struct); ok {
						fmt.Printf("%s is struct.\n", obj.Name())
						if tg.structs[obj.Pkg().Path()] == nil {
							tg.structs[obj.Pkg().Path()] = map[string]string{}
						}
						tg.structs[obj.Pkg().Path()][obj.Name()] = obj.Name()
						parent = obj
					} else if _, ok := obj.Type().Underlying().(*types.Interface); ok {
						fmt.Printf("%s is interface.\n", obj.Name())
						if tg.interfaces[obj.Pkg().Path()] == nil {
							tg.interfaces[obj.Pkg().Path()] = map[string]string{}
						}
						tg.interfaces[obj.Pkg().Path()][obj.Name()] = obj.Name()
						parent = obj
					}
					// TODO: x.TypeがStructTypeになっていて、そこから情報が取れそう。
				case *ast.StructType:
					tg.buildEdge(x.Fields.List, pkg.TypesInfo, parent, ii)
				case *ast.InterfaceType:
					tg.buildEdge(x.Methods.List, pkg.TypesInfo, parent, ii)
				}
				return true
			})
		}
	}
	return nil
}

func (tg *TypeGraph) Dump() {
	fmt.Println("structs:")
	for pkg, str := range tg.structs {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, s := range str {
			fmt.Print("    ")
			fmt.Println(s)
		}
	}

	fmt.Println("interfaces:")
	for pkg, ifc := range tg.interfaces {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, s := range ifc {
			fmt.Print("    ")
			fmt.Println(s)
		}
	}

	for _, edge := range tg.edges {
		fmt.Println(*edge)
	}
}
