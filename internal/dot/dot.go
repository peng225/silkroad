package dot

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/peng225/silkroad/internal/graph"
)

func WriteToFile(tg *graph.TypeGraph, fileName string) error {
	data := "digraph G {\n"
	data += "node[style=\"filled\" fillcolor=\"whitesmoke\"]\n"

	writeNodes := func(shape, fillColor string, nodes map[string]([]string)) {
		for pkg, objs := range nodes {
			sanitizedPkg := strings.Replace(
				strings.Replace(
					strings.Replace(pkg, ".", "_", -1),
					"/", "_", -1),
				"-", "_", -1)
			data += fmt.Sprintf("subgraph cluster_%s {\n", sanitizedPkg)
			data += fmt.Sprintf("label = \"%s\";\n", pkg)
			data += "style = \"solid\";\n"
			data += "bgcolor = \"cornsilk\";\n"
			for _, obj := range objs {
				data += fmt.Sprintf("\"%s.%s\" [label=\"%s\" shape=\"%s\" fillcolor=\"%s\"];\n",
					pkg, obj, obj, shape, fillColor)
			}
			data += "}\n"
		}
	}
	nodes := tg.StructNodes()
	writeNodes("rect", "paleturquoise1", nodes)
	nodes = tg.InterfaceNodes()
	writeNodes("hexagon", "plum1", nodes)
	nodes = tg.OtherNodes()
	writeNodes("ellipse", "whitesmoke", nodes)

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
