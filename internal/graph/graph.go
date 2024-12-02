package graph

import (
	"fmt"
	"go/ast"
	"go/types"
	"log/slog"
	"strings"

	"golang.org/x/tools/go/packages"
)

type TypeGraph struct {
	pkgToStructs    map[string](map[string]types.Object)
	pkgToInterfaces map[string](map[string]types.Object)
	pkgToOthers     map[string](map[string]types.Object)
	edges           map[string](map[Edge]struct{})
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
		edges:           map[string](map[Edge]struct{}){},
		ignoreExternal:  ignoreExternal,
		moduleName:      moduleName,
	}
}

func (tg *TypeGraph) findTypeStringsFromExpr(expr ast.Expr, info *types.Info, tps map[string]struct{}) []string {
	ret := []string{}

	switch v := expr.(type) {
	case *ast.Ident:
		obj := info.ObjectOf(v)
		if obj == nil {
			slog.Error("Obj is nil.", "name", v.Name)
			return nil
		}
		t := obj.Type()
		switch ut := t.Underlying().(type) {
		case *types.Struct:
			ret = append(ret, types.ExprString(expr))
		case *types.Interface:
			if _, ok := tps[types.ExprString(expr)]; ok {
				return nil
			}
			ret = append(ret, types.ExprString(expr))
		case *types.Basic:
			// Aliased?
			// Examples of non-aliased type: int, uint8, string
			if t.String() != ut.String() {
				ret = append(ret, types.ExprString(expr))
			}
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
			if t.String() != ut.String() {
				ret = append(ret, types.ExprString(expr))
			}
		case *types.Chan:
			// Aliased?
			if t.String() != ut.String() {
				ret = append(ret, types.ExprString(expr))
			}
		case *types.Signature:
			// Aliased?
			if t.String() != ut.String() {
				ret = append(ret, types.ExprString(expr))
			}
		default:
			slog.Warn("ut did not match any types.", "ut", ut.String(),
				"t", t.String(),
				"type", fmt.Sprintf("%T", ut))
		}
	case *ast.StarExpr:
		ret = append(ret, tg.findTypeStringsFromExpr(v.X, info, tps)...)
	case *ast.ArrayType:
		ret = append(ret, tg.findTypeStringsFromExpr(v.Elt, info, tps)...)
	case *ast.MapType:
		ret = append(ret, tg.findTypeStringsFromExpr(v.Key, info, tps)...)
		ret = append(ret, tg.findTypeStringsFromExpr(v.Value, info, tps)...)
	case *ast.SelectorExpr:
		ret = append(ret, types.ExprString(v))
	case *ast.ChanType:
		ret = append(ret, tg.findTypeStringsFromExpr(v.Value, info, tps)...)
	case *ast.FuncType:
		if v.Params != nil {
			for _, param := range v.Params.List {
				ret = append(ret, tg.findTypeStringsFromExpr(param.Type, info, tps)...)
			}
		}
		if v.Results != nil {
			for _, param := range v.Results.List {
				ret = append(ret, tg.findTypeStringsFromExpr(param.Type, info, tps)...)
			}
		}
	case *ast.IndexExpr:
		ret = append(ret, tg.findTypeStringsFromExpr(v.Index, info, tps)...)
		ret = append(ret, tg.findTypeStringsFromExpr(v.X, info, tps)...)
	case *ast.Ellipsis:
		ret = append(ret, tg.findTypeStringsFromExpr(v.Elt, info, tps)...)
	case *ast.StructType:
		// Ignore.
		// e.g. struct{}
	case *ast.InterfaceType:
		// Ignore.
		// e.g. interface{}
	default:
		slog.Warn("expr did not match any types.", "expr", types.ExprString(expr),
			"type", fmt.Sprintf("%T", v))
	}
	return ret
}

func (tg *TypeGraph) addToEdges(from, to string, kind EdgeKind) {
	if _, ok := tg.edges[from]; !ok {
		tg.edges[from] = map[Edge]struct{}{}
	}
	tg.edges[from][Edge{
		To:   to,
		Kind: kind,
	}] = struct{}{}
}

func containedInBlacklist(name string) bool {
	return name == "any" || name == "error" || name == "comparable"
}

func (tg *TypeGraph) findFullTypeName(name string, parent types.Object, ii []importInfo) string {
	fullName := parent.Pkg().Path() + "." + name
	for _, v := range ii {
		tokens := strings.Split(name, ".")
		if len(tokens) == 2 {
			if v.alias == tokens[0] || strings.HasSuffix(v.path, tokens[0]) {
				fullName = v.path + "." + tokens[1]
				break
			}
		}
	}
	return fullName
}

func (tg *TypeGraph) buildHasEdge(fields []*ast.Field, info *types.Info, parent types.Object,
	ii []importInfo, tps map[string]struct{}) {
	for _, field := range fields {
		typeNames := tg.findTypeStringsFromExpr(field.Type, info, tps)
		embedded := field.Names == nil
		kind := Has
		if embedded {
			kind = Embeds
		}
		for _, name := range typeNames {
			if containedInBlacklist(name) {
				continue
			}
			fullName := tg.findFullTypeName(name, parent, ii)
			if tg.ignoreExternal && !strings.HasPrefix(fullName, tg.moduleName) {
				continue
			}
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(), fullName, kind)
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
						tg.addToEdges(spkg+"."+s.Name(), ipkg+"."+i.Name(), Implements)
					}
				}
			}
		}
	}
}

func (tg *TypeGraph) buildEdge(x *ast.TypeSpec, info *types.Info,
	parent types.Object, ii []importInfo) {
	// Get a type parameter list
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
		tg.buildHasEdge(t.Fields.List, info, parent, ii, tps)
	case *ast.InterfaceType:
		tg.buildHasEdge(t.Methods.List, info, parent, ii, tps)
	case *ast.Ident:
		childObj := info.ObjectOf(t)
		if childObj == nil {
			return
		}
		if containedInBlacklist(childObj.Name()) {
			break
		}
		switch childObj.Type().Underlying().(type) {
		case *types.Struct:
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(childObj.Name(), parent, ii),
				UsesAsAlias)
		case *types.Interface:
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(childObj.Name(), parent, ii),
				UsesAsAlias)
		case *types.Basic:
			// Ignore.
		default:
			slog.Error("Failed to build edge", "type", fmt.Sprintf("%T", t),
				"childObjType", childObj.Type().Underlying())
		}
	case *ast.MapType:
		typs := tg.findTypeStringsFromExpr(t.Key, info, tps)
		for _, typ := range typs {
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(typ, parent, ii), UsesAsAlias)
		}
		typs = tg.findTypeStringsFromExpr(t.Value, info, tps)
		for _, typ := range typs {
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(typ, parent, ii), UsesAsAlias)
		}
	case *ast.ArrayType:
		typs := tg.findTypeStringsFromExpr(t.Elt, info, tps)
		for _, typ := range typs {
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(typ, parent, ii), UsesAsAlias)
		}
	case *ast.StarExpr:
		typs := tg.findTypeStringsFromExpr(t.X, info, tps)
		for _, typ := range typs {
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(typ, parent, ii), UsesAsAlias)
		}
	case *ast.ChanType:
		typs := tg.findTypeStringsFromExpr(t.Value, info, tps)
		for _, typ := range typs {
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(typ, parent, ii), UsesAsAlias)
		}
	case *ast.FuncType:
		typs := []string{}
		if t.Params != nil {
			for _, param := range t.Params.List {
				typs = append(typs, tg.findTypeStringsFromExpr(param.Type, info, tps)...)
			}
		}
		if t.Results != nil {
			for _, param := range t.Results.List {
				typs = append(typs, tg.findTypeStringsFromExpr(param.Type, info, tps)...)
			}
		}
		for _, typ := range typs {
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(typ, parent, ii), UsesAsAlias)
		}
	case *ast.Ellipsis:
		typs := tg.findTypeStringsFromExpr(t.Elt, info, tps)
		for _, typ := range typs {
			tg.addToEdges(parent.Pkg().Path()+"."+parent.Name(),
				tg.findFullTypeName(typ, parent, ii), UsesAsAlias)
		}
	default:
		slog.Error("Failed to build edge", "type", fmt.Sprintf("%T", t))
	}
}

func addToNodesHelper(dest map[string](map[string]types.Object), obj types.Object) {
	if dest[obj.Pkg().Path()] == nil {
		dest[obj.Pkg().Path()] = map[string]types.Object{}
	}
	dest[obj.Pkg().Path()][obj.Name()] = obj
}

// When obj is added to the node list, return true.
func (tg *TypeGraph) addToNodes(obj types.Object) bool {
	switch ut := obj.Type().Underlying().(type) {
	case *types.Struct:
		addToNodesHelper(tg.pkgToStructs, obj)
	case *types.Interface:
		addToNodesHelper(tg.pkgToInterfaces, obj)
	case *types.Basic:
		// Basic type and not aliased? (e.g. int, uint8, string)
		if obj.Type().String() == ut.String() {
			return false
		}
		addToNodesHelper(tg.pkgToOthers, obj)
	case *types.Map:
		addToNodesHelper(tg.pkgToOthers, obj)
	case *types.Slice:
		addToNodesHelper(tg.pkgToOthers, obj)
	case *types.Array:
		addToNodesHelper(tg.pkgToOthers, obj)
	case *types.Pointer:
		addToNodesHelper(tg.pkgToOthers, obj)
	case *types.Chan:
		addToNodesHelper(tg.pkgToOthers, obj)
	case *types.Signature:
		addToNodesHelper(tg.pkgToOthers, obj)
	default:
		slog.Info("obj was not added to the node list.", "name", obj.Name())
		return false
	}

	return true
}

func (tg *TypeGraph) Build(path string) error {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:  path,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}
	n := packages.PrintErrors(pkgs)
	if n != 0 {
		return fmt.Errorf("error count is not 0: %d", n)
	}

	for _, pkg := range pkgs {
		for _, syntax := range pkg.Syntax {
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
					added := tg.addToNodes(obj)
					if !added {
						return true
					}

					tg.buildEdge(x, pkg.TypesInfo, obj, ii)
				}
				return true
			})
		}
	}

	tg.buildImplementsEdge()

	return nil
}

func (tg *TypeGraph) StructNodes() map[string]([]string) {
	nodes := map[string]([]string){}

	for pkg, structs := range tg.pkgToStructs {
		nodes[pkg] = []string{}
		for _, s := range structs {
			nodes[pkg] = append(nodes[pkg], s.Name())
		}
	}

	return nodes
}

func (tg *TypeGraph) InterfaceNodes() map[string]([]string) {
	nodes := map[string]([]string){}

	for pkg, interfaces := range tg.pkgToInterfaces {
		if _, ok := nodes[pkg]; !ok {
			nodes[pkg] = []string{}
		}
		for _, i := range interfaces {
			nodes[pkg] = append(nodes[pkg], i.Name())
		}
	}

	return nodes
}

func (tg *TypeGraph) OtherNodes() map[string]([]string) {
	nodes := map[string]([]string){}

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

func (tg *TypeGraph) Edges() map[string](map[Edge]struct{}) {
	ret := map[string](map[Edge]struct{}){}
	for from, edges := range tg.edges {
		ret[from] = map[Edge]struct{}{}
		for edge, _ := range edges {
			ret[from][edge] = struct{}{}
		}
	}
	return ret
}

func (tg *TypeGraph) Dump() {
	fmt.Println("struct nodes:")
	for pkg, str := range tg.pkgToStructs {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, s := range str {
			fmt.Print("    ")
			fmt.Println(s.Name())
		}
	}

	fmt.Println("interface nodes:")
	for pkg, ifc := range tg.pkgToInterfaces {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, s := range ifc {
			fmt.Print("    ")
			fmt.Println(s.Name())
		}
	}

	fmt.Println("other nodes:")
	for pkg, others := range tg.pkgToOthers {
		fmt.Printf("  pkg: %s\n", pkg)
		for _, o := range others {
			fmt.Print("    ")
			fmt.Println(o.Name())
		}
	}

	fmt.Println("edges:")
	for from, edges := range tg.edges {
		fmt.Printf("  from: %s\n", from)
		fmt.Println("  to, kind:")
		for edge, _ := range edges {
			fmt.Printf("    %s, %d\n", edge.To, edge.Kind)
		}
	}
}
