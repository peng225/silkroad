package dot

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/peng225/silkroad/internal/graph"
)

func OutputDotFile(tg *graph.TypeGraph, fileName string) error {
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}

	err = writeAll(f, []byte("digraph G {\n"))
	if err != nil {
		return err
	}

	nodes := tg.Nodes()
	for pkg, objs := range nodes {
		sanitizedPkg := strings.Replace(
			strings.Replace(pkg, ".", "_", -1),
			"/", "_", -1)
		err := writeAll(f, []byte(fmt.Sprintf("subgraph cluster_%s {\n", sanitizedPkg)))
		if err != nil {
			return err
		}
		err = writeAll(f, []byte(fmt.Sprintf("label = \"%s\";\n", pkg)))
		if err != nil {
			return err
		}
		err = writeAll(f, []byte("style = \"solid\";\n"))
		if err != nil {
			return err
		}
		err = writeAll(f, []byte("color = \"black\";\n"))
		if err != nil {
			return err
		}
		for _, obj := range objs {
			err := writeAll(f, []byte(fmt.Sprintf("\"%s.%s\" [label = \"%s\"];\n", pkg, obj, obj)))
			if err != nil {
				return err
			}
		}
		err = writeAll(f, []byte("}\n"))
		if err != nil {
			return err
		}
	}

	for _, edge := range tg.Edges() {
		label := ""
		switch edge.Kind {
		case graph.Has:
			label = "Has"
		case graph.Implements:
			label = "Implements"
		case graph.Embeds:
			label = "Embeds"
		default:
			panic(fmt.Sprintf("Unknown edge kind %d. found", edge.Kind))
		}
		err := writeAll(f, []byte(fmt.Sprintf("\"%s\" -> \"%s\" [label = \"%s\"];\n", edge.From, edge.To, label)))
		if err != nil {
			return err
		}
	}

	err = writeAll(f, []byte("}\n"))
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
