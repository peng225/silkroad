package cmd

import (
	"bufio"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/peng225/silkroad/internal/dot"
	"github.com/peng225/silkroad/internal/graph"
	"github.com/spf13/cobra"
)

var (
	rootPath        string
	outputFileName  string
	ignoreExternal  bool
	goModPath       string
	packagePatterns []string
	verbose         bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "silkroad",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		moduleName := ""
		var err error
		if ignoreExternal {
			moduleName, err = getModuleName(path.Join(goModPath, "go.mod"))
			if err != nil {
				panic(err)
			}
		}
		tg := graph.NewTypeGraph(ignoreExternal, moduleName)
		err = tg.Build(rootPath)
		if err != nil {
			panic(err)
		}
		if verbose {
			tg.Dump()
		}
		err = dot.WriteToFile(tg, outputFileName)
		if err != nil {
			slog.Error("Failed to output a dot file.", "err", err.Error())
		}
	},
}

func getModuleName(goModFilePath string) (string, error) {
	f, err := os.Open(goModFilePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			tokens := strings.Split(line, " ")
			return tokens[len(tokens)-1], nil
		}
	}
	return "", nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.silkroad.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&rootPath, "path", "p", ".", "The path to the root directory for which the analysis runs.")
	rootCmd.Flags().StringVarP(&outputFileName, "output", "o", ".", "The output dot file name.")
	rootCmd.Flags().BoolVar(&ignoreExternal, "ignore-external", false, "Ignore types imported from the external modules.")
	rootCmd.Flags().StringVar(&goModPath, "go-mod-path", "", "The path to the directory where go.mod file exists.")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose mode.")
	rootCmd.Flags().StringSliceVar(&packagePatterns, "package-pattern", []string{"./..."}, "Package patterns. e.g. 'bytes,unicode...'")

	rootCmd.MarkFlagsRequiredTogether("ignore-external", "go-mod-path")
}
