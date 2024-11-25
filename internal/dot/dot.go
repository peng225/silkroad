package dot

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peng225/silkroad/internal/graph"
)

func WriteToFile(tg *graph.TypeGraph, fileName string) error {
	data := "digraph G {\n"

	nodes := tg.Nodes()
	for pkg, objs := range nodes {
		sanitizedPkg := strings.Replace(
			strings.Replace(
				strings.Replace(pkg, ".", "_", -1),
				"/", "_", -1),
			"-", "_", -1)
		data += fmt.Sprintf("subgraph cluster_%s {\n", sanitizedPkg)
		data += fmt.Sprintf("label = \"%s\";\n", pkg)
		data += "style = \"solid\";\n"
		data += "color = \"black\";\n"
		for _, obj := range objs {
			data += fmt.Sprintf("\"%s.%s\" [label = \"%s\"];\n", pkg, obj, obj)
		}
		data += "}\n"
	}

	for from, edges := range tg.Edges() {
		for edge, _ := range edges {
			label := ""
			switch edge.Kind {
			case graph.Has:
				label = "Has"
			case graph.Implements:
				label = "Implements"
			case graph.Embeds:
				label = "Embeds"
			case graph.UsesAsAlias:
				label = "UsesAsAlias"
			default:
				panic(fmt.Sprintf("Unknown edge kind %d. found", edge.Kind))
			}
			data += fmt.Sprintf("\"%s\" -> \"%s\" [label = \"%s\"];\n", from, edge.To, label)
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
