package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type TypeGraph struct {
	pkgToStructs    map[string](map[string]types.Object)
	pkgToInterfaces map[string](map[string]types.Object)
	edges           []*edge
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
		pkgToStructs:    map[string](map[string]types.Object){},
		pkgToInterfaces: map[string](map[string]types.Object){},
		edges:           []*edge{},
	}
}

func (tg *TypeGraph) handleExpr(expr ast.Expr, info *types.Info) []string {
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
		ret = append(ret, tg.handleExpr(v.X, info)...)
	case *ast.ArrayType:
		ret = append(ret, tg.handleExpr(v.Elt, info)...)
	case *ast.MapType:
		ret = append(ret, tg.handleExpr(v.Key, info)...)
		ret = append(ret, tg.handleExpr(v.Value, info)...)
	case *ast.SelectorExpr:
		ret = append(ret, types.ExprString(v))
	}
	return ret
}

func (tg *TypeGraph) buildHasEdge(fields []*ast.Field, info *types.Info, parent types.Object, ii []importInfo) {
	for _, field := range fields {
		// TODO: ST2 -> ST3 という組が2つできてしまう。
		// Map じゃないが、どうにか重複排除したい。
		strOrInterfaceNames := tg.handleExpr(field.Type, info)
		embedded := field.Names == nil
		kind := Has
		if embedded {
			kind = Embeds
		}
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
				kind: kind,
			})
		}
	}
}

func (tg *TypeGraph) buildImplementsEdge() {
	for ipkg, interfaces := range tg.pkgToInterfaces {
		for _, i := range interfaces {
			fmt.Printf("i: %s\n", i.Name())
			typedI, ok := i.Type().Underlying().(*types.Interface)
			if !ok {
				panic("should be interface type")
			}
			for spkg, structs := range tg.pkgToStructs {
				for _, s := range structs {
					ptr := types.NewPointer(s.Type())
					if types.Implements(ptr, typedI) {
						tg.edges = append(tg.edges, &edge{
							from: spkg + "." + s.Name(),
							to:   ipkg + "." + i.Name(),
							kind: Implements,
						})
					}
				}
			}
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
						if tg.pkgToStructs[obj.Pkg().Path()] == nil {
							tg.pkgToStructs[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToStructs[obj.Pkg().Path()][obj.Name()] = obj
					} else if _, ok := obj.Type().Underlying().(*types.Interface); ok {
						if tg.pkgToInterfaces[obj.Pkg().Path()] == nil {
							tg.pkgToInterfaces[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToInterfaces[obj.Pkg().Path()][obj.Name()] = obj
					} else {
						break
					}
					switch t := x.Type.(type) {
					case *ast.StructType:
						tg.buildHasEdge(t.Fields.List, pkg.TypesInfo, obj, ii)
					case *ast.InterfaceType:
						tg.buildHasEdge(t.Methods.List, pkg.TypesInfo, obj, ii)
					}
				}
				return true
			})
		}
	}

	tg.buildImplementsEdge()

	return nil
}

func (tg *TypeGraph) Dump() {
	fmt.Println("structs:")
	for pkg, str := range tg.pkgToStructs {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, s := range str {
			fmt.Print("    ")
			fmt.Println(s.Name())
		}
	}

	fmt.Println("interfaces:")
	for pkg, ifc := range tg.pkgToInterfaces {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, s := range ifc {
			fmt.Print("    ")
			fmt.Println(s.Name())
		}
	}

	for _, edge := range tg.edges {
		fmt.Println(*edge)
	}
}
