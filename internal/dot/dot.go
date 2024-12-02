package dot

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/peng225/silkroad/internal/graph"
)

type nodeStyle struct {
	shape     string
	fillColor string
}

type nodesWithStyle struct {
	nodes []string
	ns    nodeStyle
}

func WriteToFile(tg *graph.TypeGraph, fileName string) error {
	data := "digraph G {\n"
	data += "node[style=\"filled\" fillcolor=\"whitesmoke\"]\n"

	pkgToNodesWithStyleList := make(map[string]([]nodesWithStyle))
	for pkg, nodes := range tg.StructNodes() {
		pkgToNodesWithStyleList[pkg] = []nodesWithStyle{
			{
				nodes: nodes,
				ns: nodeStyle{
					shape:     "rect",
					fillColor: "paleturquoise1",
				},
			},
		}
	}
	for pkg, nodes := range tg.InterfaceNodes() {
		pkgToNodesWithStyleList[pkg] = append(pkgToNodesWithStyleList[pkg],
			nodesWithStyle{
				nodes: nodes,
				ns: nodeStyle{
					shape:     "hexagon",
					fillColor: "plum1",
				},
			})
	}
	for pkg, nodes := range tg.OtherNodes() {
		pkgToNodesWithStyleList[pkg] = append(pkgToNodesWithStyleList[pkg],
			nodesWithStyle{
				nodes: nodes,
				ns: nodeStyle{
					shape:     "ellipse",
					fillColor: "whitesmoke",
				},
			})
	}
	for pkg, nwsList := range pkgToNodesWithStyleList {
		sanitizedPkg := strings.Replace(
			strings.Replace(
				strings.Replace(pkg, ".", "_", -1),
				"/", "_", -1),
			"-", "_", -1)
		data += fmt.Sprintf("subgraph cluster_%s {\n", sanitizedPkg)
		data += fmt.Sprintf("  label = \"%s\";\n", pkg)
		data += "  style = \"solid\";\n"
		data += "  bgcolor = \"cornsilk\";\n"
		for _, nws := range nwsList {
			for _, obj := range nws.nodes {
				data += fmt.Sprintf("  \"%s.%s\" [label=\"%s\" shape=\"%s\" fillcolor=\"%s\"];\n",
					pkg, obj, obj, nws.ns.shape, nws.ns.fillColor)
			}
		}
		data += "}\n"
	}

	for from, edges := range tg.Edges() {
		for edge, _ := range edges {
			label := ""
			arrowHead := "normal"
			style := "solid"
			switch edge.Kind {
			case graph.Has:
				label = "Has"
			case graph.Implements:
				label = "Implements"
				arrowHead = "empty"
				style = "dashed"
			case graph.Embeds:
				label = "Embeds"
				arrowHead = "empty"
			case graph.UsesAsAlias:
				label = "UsesAsAlias"
				style = "dashed"
			default:
				slog.Warn("Unknown edge kind found", "kind", edge.Kind)
			}
			data += fmt.Sprintf("\"%s\" -> \"%s\" [label=\"%s\" arrowhead=\"%s\" style=\"%s\"];\n",
				from, edge.To, label, arrowHead, style)
		}
	}
	data += "}\n"

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	err = writeAll(f, []byte(data))
	if err != nil {
		return err
	}

	return nil
}

func writeAll(r io.Writer, data []byte) error {
	tmpData := data
	for len(tmpData) != 0 {
		n, err := r.Write(tmpData)
		if err != nil {
			return err
		}
		tmpData = tmpData[n:]
	}
	return nil
}
