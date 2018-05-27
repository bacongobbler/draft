package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"

	"github.com/spf13/cobra"
)

type pluginUninstallCmd struct {
	names []string
	home  draftpath.Home
	out   io.Writer
}

func newPluginUninstallCmd(out io.Writer) *cobra.Command {
	pcmd := &pluginUninstallCmd{out: out}
	cmd := &cobra.Command{
		Use:   "uninstall <plugin>...",
		Short: "uninstall one or more Draft plugins",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return pcmd.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return pcmd.run()
		},
	}
	return cmd
}

func (pcmd *pluginUninstallCmd) complete(args []string) error {
	if len(args) == 0 {
		return errors.New("please provide plugin name to remove")
	}
	pcmd.names = args
	pcmd.home = draftpath.Home(homePath())
	return nil
}

func (pcmd *pluginUninstallCmd) run() error {
	pHome := plugin.Home(pcmd.home.Plugins())
	for _, pluginName := range pcmd.names {
		installedPlugins := findInstalledPlugins(pHome)
		switch len(installedPlugins) {
		case 0:
			return fmt.Errorf("no plugin with the name '%s' was found", pluginName)
		case 1:
			pluginName = installedPlugins[0]
		default:
			var match bool
			// check if we have an exact match
			for _, f := range installedPlugins {
				if strings.Compare(f, pluginName) == 0 {
					match = true
				}
			}
			if !match {
				return fmt.Errorf("%d plugins with the name '%s' was found: %v", len(installedPlugins), pluginName, installedPlugins)
			}
		}
		p := plugin.Plugin{
			Name: pluginName,
		}
		fmt.Fprintf(pcmd.out, "Uninstalling %s...\n", p.Name)
		start := time.Now()
		if err := p.Uninstall(pHome); err != nil {
			return err
		}
		t := time.Now()
		fmt.Fprintf(pcmd.out, "%s: uninstalled in %s\n", p.Name, t.Sub(start).String())
		return nil
	}
	return nil
}
