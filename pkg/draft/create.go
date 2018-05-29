package draft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Azure/draft/pkg/draft/manifest"
	"github.com/BurntSushi/toml"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const (
	// DrakeFilename is the default filename for Drakefile.
	DrakeFilename = "Drakefile.go"
	// DraftTomlFilename is the default filename for Draft application configuration.
	DraftTomlFilename = "draft.toml"
	DraftTomlFilepath = "config/draft.toml"
	RoutesFile        = "routes"
	BinDir            = "bin"
	ConfigDir         = "config"
	LibDir            = "lib"
	LogDir            = "log"
	StaticDir         = "static"
	IndexFilename     = "index.html"
	ReadmeFilename    = "README.md"
	TestDir           = "test"
	// ChartFilename is the default Chart file name.
	ChartFilename = "Chart.yaml"
	// ValuesFilename is the default values file name.
	ValuesFilename = "values.yaml"
	// IgnoreFilename is the name of the Helm ignore file.
	IgnoreFilename = ".helmignore"
	// DeploymentFilename is the name of the deployment file.
	DeploymentFilename = "deployment.yaml"
	// ServiceFilename is the name of the service file.
	ServiceFilename = "service.yaml"
	// IngressFilename is the name of the ingress file.
	IngressFilename = "ingress.yaml"
	// NotesFilename is the name of the NOTES.txt file.
	NotesFilename = "NOTES.txt"
	// HelpersFilename is the name of the helpers file.
	HelpersFilename = "_helpers.tpl"
	// TemplatesDir is the relative directory name for templates.
	TemplatesDir = "templates"
	// ChartsDir is the directory name for the packaged chart.
	// This also doubles as the directory name for chart dependencies.
	ChartsDir        = "charts"
	DefaultDraftToml = ""
	DefaultRoutes    = ""
	DefaultIndex     = `<html><body><h1>It works!</h1>
<p>This is the default web page for Draft.</p>
<p>Draft is running, but no content has been added... Yet.</p>
</body></html>
`
	DefaultDrakefile = `// +build drake

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Default target to run when none is specified
// If not set, running 'draft task' will list available targets
// var Default = Test

// Runs 'go test ./...'.
func Test() error {
	fmt.Println("Testing...")
	cmd := exec.Command("go", "test", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
`
	DefaultReadme = `# README

This README would normally document whatever steps are necessary to get the
application up and running.

Things you may want to cover:

* Draft version

* System dependencies

* Configuration

* How to run the test suite

* Services required (job queues, cache servers, search engines, etc.)

* Deployment instructions

* ...`
)

// Create creates a new application in a directory.
//
// Create() will start by creating a directory based on the value of dir.
// It will then write the necessary files and create the appropriate directories.
//
// If draft.toml or any directories cannot be created, this will return an error.
func Create(name, dir string) error {
	path, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	for _, d := range []string{BinDir, ConfigDir, LibDir, LogDir, StaticDir, TestDir} {
		if err := os.MkdirAll(filepath.Join(path, d), 0755); err != nil {
			return err
		}
	}

	files := []struct {
		path    string
		content []byte
	}{
		{
			// Drakefile.go
			path:    filepath.Join(path, DrakeFilename),
			content: []byte(DefaultDrakefile),
		},
		{
			// README.md
			path:    filepath.Join(path, ReadmeFilename),
			content: []byte(DefaultReadme),
		},
		{
			// config/draft.toml
			path:    filepath.Join(path, ConfigDir, DraftTomlFilename),
			content: []byte(DefaultDraftToml),
		},
		{
			// config/routes
			path:    filepath.Join(path, ConfigDir, RoutesFile),
			content: []byte(DefaultRoutes),
		},
		{
			// static/index.html
			path:    filepath.Join(path, StaticDir, IndexFilename),
			content: []byte(DefaultIndex),
		},
	}

	for _, file := range files {
		if err := ioutil.WriteFile(file.path, file.content, 0644); err != nil {
			return err
		}
	}

	// Create the chart directory
	chart := &chart.Metadata{
		Name:        filepath.Base(path),
		Description: "A Helm chart for Kubernetes",
		ApiVersion:  "v1",
		Version:     "0.1.0",
	}
	chartPath := filepath.Join(path, ChartsDir)
	if err := os.Mkdir(chartPath, 0755); err != nil {
		return fmt.Errorf("Could not create %s: %s", chartPath, err)
	}
	if _, err := chartutil.Create(chart, chartPath); err != nil {
		return err
	}

	mfest := manifest.New()
	mfest.Environments[manifest.DefaultEnvironmentName].Name = name
	tomlFile := filepath.Join(path, ConfigDir, DraftTomlFilename)
	draftToml, err := os.OpenFile(tomlFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer draftToml.Close()

	if err := toml.NewEncoder(draftToml).Encode(mfest); err != nil {
		return fmt.Errorf("could not write metadata to %s: %v", tomlFile, err)
	}
	return nil
}
