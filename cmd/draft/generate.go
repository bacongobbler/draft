package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"
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
	generateCmd, _, err := baseCmd.Find([]string{"generate"})
	if err != nil {
		panic(err)
	}
	pHome := plugin.Home(home.Plugins())
	// Now we create commands for all of these.
	for _, plug := range findInstalledPlugins(pHome) {
		p, _, err := getPlugin(plug, pHome)
		if err != nil {
			log.Debugf("could not load plugin %s: %v", p, err)
			continue
		}
		var commandExists bool
		for _, command := range generateCmd.Commands() {
			if strings.Compare(command.Short, p.Description) == 0 {
				commandExists = true
			}
		}
		if commandExists {
			log.Debugf("command %s exists", p.Name)
			continue
		}

		if !strings.HasPrefix(p.Name, "generator-") {
			log.Debugf("command %s is not a generator, skipping", p.Name)
			continue
		}

		generatorName := strings.TrimPrefix(p.Name, "generator-")
		c := &cobra.Command{
			Use:   generatorName,
			Short: p.Description,
			RunE: func(cmd *cobra.Command, args []string) error {

				k, u := manuallyProcessArgs(args)
				if err := cmd.Parent().ParseFlags(k); err != nil {
					return err
				}

				// Call setupEnv before PrepareCommand because
				// PrepareCommand uses os.ExpandEnv and expects the
				// setupEnv vars.
				setupPluginEnv(generatorName, filepath.Join(pHome.Installed(), p.Name, p.Version), draftpath.Home(homePath()))
				main := filepath.Join(os.Getenv("DRAFT_PLUGIN_DIR"), p.GetPackage(runtime.GOOS, runtime.GOARCH).Path)

				prog := exec.Command(main, u...)
				prog.Env = os.Environ()
				prog.Stdout = out
				prog.Stderr = os.Stderr
				prog.Stdin = in
				return prog.Run()
			},
			// This passes all the flags to the subcommand.
			DisableFlagParsing: true,
		}

		if p.UseTunnel {
			c.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
				// Parse the parent flag, but not the local flags.
				k, _ := manuallyProcessArgs(args)
				if err := cmd.Parent().ParseFlags(k); err != nil {
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

		generateCmd.AddCommand(c)
	}
}
