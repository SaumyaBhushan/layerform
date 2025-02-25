package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/hashicorp/go-hclog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ergomake/layerform/internal/lfconfig"
	"github.com/ergomake/layerform/pkg/data"
)

func init() {
	listCmd.AddCommand(listInstancesCmd)
}

var listInstancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "List layers instances",
	Long: `List layers instances.

Prints a table of the most important information about layer instances.`,

	Run: func(_ *cobra.Command, _ []string) {
		logger := hclog.Default()
		logLevel := hclog.LevelFromString(os.Getenv("LF_LOG"))
		if logLevel != hclog.NoLevel {
			logger.SetLevel(logLevel)
		}
		ctx := hclog.WithContext(context.Background(), logger)

		cfg, err := lfconfig.Load("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", errors.Wrap(err, "fail to load config"))
			os.Exit(1)
			return
		}

		layersBackend, err := cfg.GetDefinitionsBackend(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", errors.Wrap(err, "fail to get layers backend"))
			os.Exit(1)
			return
		}

		instancesBackend, err := cfg.GetInstancesBackend(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", errors.Wrap(err, "fail to get layers instances backend"))
			os.Exit(1)
			return
		}

		instances, err := instancesBackend.ListInstances(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", errors.Wrap(err, "fail to list layer instances"))
			os.Exit(1)
			return
		}

		if len(instances) == 0 {
			fmt.Fprintln(os.Stdout, "No layer instances spawned, spawn layers by running \"layerform spawn\"")
			return
		}

		layers, err := layersBackend.ListLayers(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", errors.Wrap(err, "fail to list layer definitions"))
			os.Exit(1)
			return
		}

		layersByName := make(map[string]*data.LayerDefinition)
		for _, l := range layers {
			layersByName[l.Name] = l
		}

		sortInstancesByDepth(instances, layersByName)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintln(w, "INSTANCE NAME\tLAYER NAME\tDEPENDENCIES\tSTATUS")
		for _, instance := range instances {
			layer := layersByName[instance.DefinitionName]
			deps := ""
			for i, dep := range layer.Dependencies {
				if i > 0 {
					deps += ","
				}

				depInstName := instance.GetDependencyInstanceName(dep)
				deps += dep + "=" + depInstName
			}

			fmt.Fprintln(w, instance.InstanceName+"\t"+instance.DefinitionName+"\t"+deps+"\t"+string(instance.Status))
		}
		err = w.Flush()

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", errors.Wrap(err, "fail to print output"))
			os.Exit(1)
		}
	},
}

func computeDepth(layer *data.LayerDefinition, layers map[string]*data.LayerDefinition, level int) int {
	depth := level
	for _, d := range layer.Dependencies {
		dDepth := computeDepth(layers[d], layers, level+1)
		if dDepth > depth {
			depth = dDepth
		}
	}

	return depth
}

func sortInstancesByDepth(instances []*data.LayerInstance, layers map[string]*data.LayerDefinition) {
	sort.SliceStable(instances, func(x, y int) bool {
		instX := instances[x]
		layerX := layers[instX.DefinitionName]
		depthX := computeDepth(layerX, layers, 0)

		instY := instances[y]
		layerY := layers[instY.DefinitionName]
		depthY := computeDepth(layerY, layers, 0)

		return depthX < depthY
	})
}
