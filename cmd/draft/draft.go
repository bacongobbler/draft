// Copyright (c) Microsoft Corporation. All rights reserved.
//
// Licensed under the MIT license.

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

const (
	homeEnvVar      = "DRAFT_HOME"
	hostEnvVar      = "HELM_HOST"
	namespaceEnvVar = "TILLER_NAMESPACE"
)

var (
	// flagDebug is a signal that the user wants additional output.
	flagDebug   bool
	kubeContext string
	// draftHome depicts the home directory where all Draft config is stored.
	draftHome string
	// displayEmoji shows emoji in the console output
	displayEmoji bool
	//rootCmd is the root command handling `draft`. It's used in other parts of package cmd to add/search the command tree.
	rootCmd *cobra.Command
	// globalConfig is the configuration stored in $DRAFT_HOME/config.toml
	globalConfig DraftConfig
)

var globalUsage = `The application deployment tool for Kubernetes.
`

func init() {
	rootCmd = newRootCmd(os.Stdout, os.Stdin)
}

func newRootCmd(out io.Writer, in io.Reader) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "draft",
		Short:        globalUsage,
		Long:         globalUsage,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if flagDebug {
				log.SetLevel(log.DebugLevel)
			}
			os.Setenv(homeEnvVar, draftHome)
			globalConfig, err = ReadConfig()
			return
		},
	}
	p := cmd.PersistentFlags()
	p.StringVar(&draftHome, "home", defaultDraftHome(), "location of your Draft config. Overrides $DRAFT_HOME")
	p.BoolVar(&flagDebug, "debug", false, "enable verbose output")
	p.StringVar(&kubeContext, "kube-context", "", "name of the kubeconfig context to use when talking to Tiller")
	p.BoolVar(&displayEmoji, "display-emoji", true, "display emoji in output")

	cmd.AddCommand(
		newConfigCmd(out),
		newCreateCmd(out),
		newHomeCmd(out),
		newInitCmd(out, in),
		newUpCmd(out),
		newVersionCmd(out),
		newPluginCmd(out),
		newConnectCmd(out),
		newDeleteCmd(out),
		newLogsCmd(out),
		newHistoryCmd(out),
		newPackCmd(out),
	)

	// Find and add plugins
	loadPlugins(cmd, draftpath.Home(homePath()), out, in)

	return cmd
}

func defaultDraftHome() string {
	if home := os.Getenv(homeEnvVar); home != "" {
		return home
	}

	homeEnvPath := os.Getenv("HOME")
	if homeEnvPath == "" && runtime.GOOS == "windows" {
		homeEnvPath = os.Getenv("USERPROFILE")
	}

	return filepath.Join(homeEnvPath, ".draft")
}

func homePath() string {
	return os.ExpandEnv(draftHome)
}

// getKubeClient creates a Kubernetes config and client for a given kubeconfig context.
func getKubeClient(context string) (kubernetes.Interface, *rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return client, config, nil
}

func debug(format string, args ...interface{}) {
	if flagDebug {
		format = fmt.Sprintf("[debug] %s\n", format)
		fmt.Printf(format, args...)
	}
}

func validateArgs(args, expectedArgs []string) error {
	if len(args) != len(expectedArgs) {
		return fmt.Errorf("This command needs %v argument(s): %v", len(expectedArgs), expectedArgs)
	}
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
