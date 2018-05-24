package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

const (
	generateDesc = `Runs a generator.

Generators are used to generate boilerplate code to scaffold your application.
`
)

func newGenerateCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "run a Draft generator",
		Long:  generateDesc,
	}
	return cmd
}

// loadGenerators loads generator plugins into the command list.
//
// This follows a different pattern than loading plugins because generators
// are prefixed with `generator-` and are loaded under `draft generate <name>`
// rather than `draft <name>`.
func loadGenerators(baseCmd *cobra.Command, home draftpath.Home, out io.Writer, in io.Reader) {
	plugdirs := pluginDirPath(home)

	found, err := findPlugins(plugdirs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load plugins: %s", err)
		return
	}

	// Now we create commands for all of these.
	for _, plug := range found {
		var commandExists bool
		generateCmd, _, err := baseCmd.Find([]string{"generate"})
		if err != nil {
			panic(err)
		}
		for _, command := range generateCmd.Commands() {
			if strings.Compare(command.Use, plug.Metadata.Usage) == 0 {
				commandExists = true
			}
		}
		if commandExists {
			log.Debugf("command %s exists", plug.Metadata.Usage)
			continue
		}
		if !strings.HasPrefix(plug.Metadata.Name, "generator-") {
			log.Debugf("command %s is NOT a generator, skipping", plug.Metadata.Name)
			continue
		}
		plug := plug
		md := plug.Metadata
		if md.Usage == "" {
			md.Usage = fmt.Sprintf("the %q generator", md.Name)
		}

		c := &cobra.Command{
			Use:   md.Name,
			Short: md.Usage,
			Long:  md.Description,
			RunE: func(cmd *cobra.Command, args []string) error {

				k, u := manuallyProcessArgs(args)
				if err := cmd.Parent().ParseFlags(k); err != nil {
					return err
				}

				// Call setupEnv before PrepareCommand because
				// PrepareCommand uses os.ExpandEnv and expects the
				// setupEnv vars.
				setupPluginEnv(md.Name, plug.Metadata.Version, plug.Dir, plugdirs, draftpath.Home(homePath()))
				main, argv := plug.PrepareCommand(u)

				prog := exec.Command(main, argv...)
				prog.Env = os.Environ()
				prog.Stdout = out
				prog.Stderr = os.Stderr
				prog.Stdin = in
				return prog.Run()
			},
			// This passes all the flags to the subcommand.
			DisableFlagParsing: true,
		}
		if md.UseTunnel {
			c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
				// Parse the parent flag, but not the local flags.
				k, _ := manuallyProcessArgs(args)
				if err := c.Parent().ParseFlags(k); err != nil {
					return err
				}
				client, config, err := getKubeClient(kubeContext)
				if err != nil {
					return fmt.Errorf("Could not get a kube client: %s", err)
				}

				tillerTunnel, err := setupTillerConnection(client, config, tillerNamespace)
				if err != nil {
					return err
				}
				tillerHost = fmt.Sprintf("127.0.0.1:%d", tillerTunnel.Local)
				return nil
			}
		}

		baseCmd.AddCommand(c)
	}
}
