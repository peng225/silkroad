package graph

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
	pkgToOthers     map[string](map[string]types.Object)
	edges           []*Edge
	ignoreExternal  bool
	moduleName      string
}

type EdgeKind int

const (
	Has EdgeKind = iota
	Implements
	Embeds
	UsesAsAlias
)

type Edge struct {
	From string
	To   string
	Kind EdgeKind
}

type importInfo struct {
	alias string
	path  string
}

func NewTypeGraph(ignoreExternal bool, moduleName string) *TypeGraph {
	return &TypeGraph{
		pkgToStructs:    map[string](map[string]types.Object){},
		pkgToInterfaces: map[string](map[string]types.Object){},
		pkgToOthers:     map[string](map[string]types.Object){},
		edges:           []*Edge{},
		ignoreExternal:  ignoreExternal,
		moduleName:      moduleName,
	}
}

func (tg *TypeGraph) findTypeStringsFromExpr(expr ast.Expr, info *types.Info, tps map[string]struct{}) []string {
	ret := []string{}

	t := info.TypeOf(expr)
	if t == nil {
		return nil
	}
	switch ut := t.Underlying().(type) {
	case *types.Struct:
		ret = append(ret, types.ExprString(expr))
		return ret
	case *types.Interface:
		if _, ok := tps[types.ExprString(expr)]; ok {
			return nil
		}
		ret = append(ret, types.ExprString(expr))
		return ret
	case *types.Basic:
		// Not aliased? (e.g. int, uint8, string)
		if t.String() == ut.String() {
			return nil
		}
		ret = append(ret, types.ExprString(expr))
		return ret
	case *types.Map:
		// Aliased?
		if t.String() != ut.String() {
			ret = append(ret, types.ExprString(expr))
		}
	case *types.Slice:
		// Aliased?
		if t.String() != ut.String() {
			ret = append(ret, types.ExprString(expr))
		}
	case *types.Array:
		// Aliased?
		if t.String() != ut.String() {
			ret = append(ret, types.ExprString(expr))
		}
	case *types.Pointer:
		// Aliased?
		fmt.Printf("pointer, underlying: %s, %s\n", t.String(), ut.String())
		if t.String() != ut.String() {
			ret = append(ret, types.ExprString(expr))
		}
	}

	switch v := expr.(type) {
	case *ast.StarExpr:
		ret = append(ret, tg.findTypeStringsFromExpr(v.X, info, tps)...)
	case *ast.ArrayType:
		ret = append(ret, tg.findTypeStringsFromExpr(v.Elt, info, tps)...)
	case *ast.MapType:
		ret = append(ret, tg.findTypeStringsFromExpr(v.Key, info, tps)...)
		ret = append(ret, tg.findTypeStringsFromExpr(v.Value, info, tps)...)
	case *ast.SelectorExpr:
		ret = append(ret, types.ExprString(v))
	}
	return ret
}

func (tg *TypeGraph) buildHasEdge(fields []*ast.Field, info *types.Info, parent types.Object,
	ii []importInfo, tps map[string]struct{}) {
	for _, field := range fields {
		// TODO: ST2 -> ST3 という組が2つできてしまう。
		// Map じゃないが、どうにか重複排除したい。
		typeNames := tg.findTypeStringsFromExpr(field.Type, info, tps)
		embedded := field.Names == nil
		kind := Has
		if embedded {
			kind = Embeds
		}
	TYPES_LOOP:
		for _, name := range typeNames {
			if name == "struct{}" || name == "interface{}" || name == "any" {
				continue
			}
			fullName := parent.Pkg().Path() + "." + name
			for _, v := range ii {
				tokens := strings.Split(name, ".")
				if len(tokens) == 2 {
					if v.alias == tokens[0] || strings.HasSuffix(v.path, tokens[0]) {
						fullName = v.path + "." + tokens[1]
						if tg.ignoreExternal && !strings.HasPrefix(fullName, tg.moduleName) {
							continue TYPES_LOOP
						}
						break
					}
				}
			}
			tg.edges = append(tg.edges, &Edge{
				From: parent.Pkg().Path() + "." + parent.Name(),
				To:   fullName,
				Kind: kind,
			})
		}
	}
}

func (tg *TypeGraph) buildImplementsEdge() {
	for ipkg, interfaces := range tg.pkgToInterfaces {
		for _, i := range interfaces {
			typedI, ok := i.Type().Underlying().(*types.Interface)
			if !ok {
				panic("should be interface type")
			}
			for spkg, structs := range tg.pkgToStructs {
				for _, s := range structs {
					ptr := types.NewPointer(s.Type())
					if types.Implements(ptr, typedI) && !typedI.Empty() {
						tg.edges = append(tg.edges, &Edge{
							From: spkg + "." + s.Name(),
							To:   ipkg + "." + i.Name(),
							Kind: Implements,
						})
					}
				}
			}
		}
	}
}

func (tg *TypeGraph) buildAliasEdge(pkg, from, to string) {
	tg.edges = append(tg.edges, &Edge{
		From: pkg + "." + from,
		To:   pkg + "." + to,
		Kind: UsesAsAlias,
	})
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
					switch ut := obj.Type().Underlying().(type) {
					case *types.Struct:
						if tg.pkgToStructs[obj.Pkg().Path()] == nil {
							tg.pkgToStructs[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToStructs[obj.Pkg().Path()][obj.Name()] = obj
					case *types.Interface:
						if tg.pkgToInterfaces[obj.Pkg().Path()] == nil {
							tg.pkgToInterfaces[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToInterfaces[obj.Pkg().Path()][obj.Name()] = obj
					case *types.Basic:
						// Basic type and not aliased? (e.g. int, uint8, string)
						if obj.Type().String() == ut.String() {
							return true
						}
						if tg.pkgToOthers[obj.Pkg().Path()] == nil {
							tg.pkgToOthers[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToOthers[obj.Pkg().Path()][obj.Name()] = obj
					case *types.Map:
						if tg.pkgToOthers[obj.Pkg().Path()] == nil {
							tg.pkgToOthers[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToOthers[obj.Pkg().Path()][obj.Name()] = obj
					case *types.Slice:
						if tg.pkgToOthers[obj.Pkg().Path()] == nil {
							tg.pkgToOthers[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToOthers[obj.Pkg().Path()][obj.Name()] = obj
					case *types.Array:
						if tg.pkgToOthers[obj.Pkg().Path()] == nil {
							tg.pkgToOthers[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToOthers[obj.Pkg().Path()][obj.Name()] = obj
					case *types.Pointer:
						if tg.pkgToOthers[obj.Pkg().Path()] == nil {
							tg.pkgToOthers[obj.Pkg().Path()] = map[string]types.Object{}
						}
						tg.pkgToOthers[obj.Pkg().Path()][obj.Name()] = obj
					default:
						return true
					}

					tps := map[string]struct{}{}
					if x.TypeParams != nil {
						for _, tp := range x.TypeParams.List {
							for _, name := range tp.Names {
								tps[name.Name] = struct{}{}
							}
						}
					}
					switch t := x.Type.(type) {
					case *ast.StructType:
						tg.buildHasEdge(t.Fields.List, pkg.TypesInfo, obj, ii, tps)
					case *ast.InterfaceType:
						tg.buildHasEdge(t.Methods.List, pkg.TypesInfo, obj, ii, tps)
					case *ast.Ident:
						childObj := pkg.TypesInfo.ObjectOf(t)
						if childObj == nil {
							return true
						}
						switch childObj.Type().Underlying().(type) {
						case *types.Struct:
							tg.buildAliasEdge(obj.Pkg().Path(), obj.Name(), childObj.Name())
						}
					case *ast.MapType:
						typs := tg.findTypeStringsFromExpr(t.Key, pkg.TypesInfo, tps)
						for _, typ := range typs {
							tg.buildAliasEdge(obj.Pkg().Path(), obj.Name(), typ)
						}
						typs = tg.findTypeStringsFromExpr(t.Value, pkg.TypesInfo, tps)
						for _, typ := range typs {
							tg.buildAliasEdge(obj.Pkg().Path(), obj.Name(), typ)
						}
					case *ast.ArrayType:
						typs := tg.findTypeStringsFromExpr(t.Elt, pkg.TypesInfo, tps)
						for _, typ := range typs {
							tg.buildAliasEdge(obj.Pkg().Path(), obj.Name(), typ)
						}
					case *ast.StarExpr:
						typs := tg.findTypeStringsFromExpr(t.X, pkg.TypesInfo, tps)
						for _, typ := range typs {
							tg.buildAliasEdge(obj.Pkg().Path(), obj.Name(), typ)
						}
					}
				}
				return true
			})
		}
	}

	tg.buildImplementsEdge()

	return nil
}

func (tg *TypeGraph) Nodes() map[string]([]string) {
	nodes := map[string]([]string){}

	for pkg, structs := range tg.pkgToStructs {
		nodes[pkg] = []string{}
		for _, s := range structs {
			nodes[pkg] = append(nodes[pkg], s.Name())
		}
	}

	for pkg, interfaces := range tg.pkgToInterfaces {
		if _, ok := nodes[pkg]; !ok {
			nodes[pkg] = []string{}
		}
		for _, i := range interfaces {
			nodes[pkg] = append(nodes[pkg], i.Name())
		}
	}

	for pkg, others := range tg.pkgToOthers {
		if _, ok := nodes[pkg]; !ok {
			nodes[pkg] = []string{}
		}
		for _, i := range others {
			nodes[pkg] = append(nodes[pkg], i.Name())
		}
	}

	return nodes
}

func (tg *TypeGraph) Edges() []*Edge {
	ret := make([]*Edge, len(tg.edges))
	copy(ret, tg.edges)
	return ret
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

	fmt.Println("others:")
	for pkg, others := range tg.pkgToOthers {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, o := range others {
			fmt.Print("    ")
			fmt.Println(o.Name())
		}
	}

	for _, edge := range tg.edges {
		fmt.Println(*edge)
	}
}
