package main

import (
	"fmt"
	"io"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"
	"github.com/gosuri/uitable"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pluginSearchCmd struct {
	out  io.Writer
	home plugin.Home
}

func newPluginSearchCmd(out io.Writer) *cobra.Command {
	c := pluginSearchCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "search [keyword...]",
		Short: "perform a fuzzy search against available plugins",
		Run: func(cmd *cobra.Command, args []string) {
			c.home = plugin.Home(draftpath.Home(homePath()).Plugins())
			c.run(args)
		},
	}
	return cmd
}

func (pscmd *pluginSearchCmd) run(args []string) {
	foundPlugins := search(args, pscmd.home)
	table := uitable.New()
	table.AddRow("NAME", "REPOSITORY", "VERSION", "DESCRIPTION")
	for _, plugin := range foundPlugins {
		p, repository, err := getPlugin(plugin, pscmd.home)
		if err == nil {
			table.AddRow(p.Name, repository, p.Version, p.Description)
		} else {
			log.Debugln(err)
		}
	}
	fmt.Fprintln(pscmd.out, table)
}
