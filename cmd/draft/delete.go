package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"

	"github.com/Azure/draft/pkg/local"
	"github.com/Azure/draft/pkg/storage/kube/configmap"
	"github.com/Azure/draft/pkg/tasks"
)

const deleteDesc = `This command deletes an application from your Kubernetes environment.`

type deleteCmd struct {
	appName string
	out     io.Writer
}

func newDeleteCmd(out io.Writer) *cobra.Command {
	var runningEnvironment string

	dc := &deleteCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete [app]",
		Short: "delete an application",
		Long:  deleteDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				dc.appName = args[0]
			}
			return dc.run(runningEnvironment)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&runningEnvironment, environmentFlagName, environmentFlagShorthand, defaultDraftEnvironment(), environmentFlagUsage)

	return cmd
}

func (d *deleteCmd) run(runningEnvironment string) error {

	var name string

	if d.appName != "" {
		name = d.appName
	} else {
		deployedApp, err := local.DeployedApplication(draftToml, runningEnvironment)
		if err != nil {
			return errors.New("Unable to detect app name\nPlease pass in the name of the application")

		}

		name = deployedApp.Name
	}

	//TODO: replace with serverside call
	if err := Delete(name); err != nil {
		return err
	}

	msg := "app '" + name + "' deleted"
	fmt.Fprintln(d.out, msg)
	return nil
}

// Delete uses the helm client to delete an app with the given name
//
// Returns an error if the command failed.
func Delete(app string) error {
	// set up helm client
	client, _, err := getKubeClient(kubeContext)
	if err != nil {
		return fmt.Errorf("Could not get a kube client: %s", err)
	}

	// delete Draft storage for app
	store := configmap.NewConfigMaps(client.CoreV1().ConfigMaps("default"))
	if _, err := store.DeleteBuilds(context.Background(), app); err != nil {
		return err
	}

	settings := cli.New()
	actionConfig := new(action.Configuration)
	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}

	uninstallAction := action.NewUninstall(actionConfig)

	// delete helm release
	_, err = uninstallAction.Run(app)
	if err != nil {
		return err
	}

	taskList, err := tasks.Load(tasksTOMLFile)
	if err != nil {
		if err == tasks.ErrNoTaskFile {
			debug(err.Error())
		} else {
			return err
		}
	} else {
		if _, err = taskList.Run(tasks.DefaultRunner, tasks.PostDelete, ""); err != nil {
			return err
		}
	}

	return nil
}
