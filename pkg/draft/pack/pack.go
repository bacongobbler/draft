package pack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/Azure/draft/pkg/osutil"
)

const (
	// PackfileName is the default Pack file name.
	PackfileName = "pack.toml"
	// ChartfileName is the default Chart file name.
	ChartfileName = "Chart.yaml"
	// ChartDir is the relative directory name for the packaged chart with a pack.
	ChartDir = "chart"
	// DockerfileName is the name of the Dockerfile.
	DockerfileName = "Dockerfile"
	// ValuesfileName is the default values file name.
	ValuesfileName = "values.yaml"
	// IgnorefileName is the name of the Helm ignore file.
	IgnorefileName = ".helmignore"
	// DeploymentName is the name of the deployment file.
	DeploymentName = "deployment.yaml"
	// ServiceName is the name of the service file.
	ServiceName = "service.yaml"
	// IngressName is the name of the ingress file.
	IngressName = "ingress.yaml"
	// NotesName is the name of the NOTES.txt file.
	NotesName = "NOTES.txt"
	// HelpersName is the name of the helpers file.
	HelpersName = "_helpers.tpl"
	// TemplatesDir is the relative directory name for templates.
	TemplatesDir = "templates"
	// ChartsDir is the relative directory name for charts dependencies.
	ChartsDir = "charts"
	// HerokuLicenseName is the name of the Neroku License
	HerokuLicenseName = "NOTICE"
	// DockerignoreName is the name of the Docker ignore file
	DockerignoreName = ".dockerignore"
)

// Pack defines a Draft Starter Pack.
type Pack struct {
	// Name is the human-readable name of the Draft pack. This is typically scoped to the
	// language's runtime configuration (e.g. "Python 3", "OpenJDK 8")
	Name string `toml:"name"`
	// Language is the programming language this pack is typically identified as when run
	// against pkg/linguist (e.g. "Python", "Java")
	Language string `toml:"language"`
	// Path is the local file path where this pack resides.
	Path string `toml:"-"`
	// Chart is the Helm chart to be installed with the Pack.
	Chart *chart.Chart `toml:"-"`
	// Dockerfile is the pre-defined Dockerfile that will be installed with the Pack.
	Dockerfile []byte `toml:"-"`
}

// SaveDir saves a pack as files in a directory.
func (p *Pack) SaveDir(dest string) error {
	// Create the chart directory
	chartPath := filepath.Join(dest, ChartDir)
	if err := os.Mkdir(chartPath, 0755); err != nil {
		return fmt.Errorf("Could not create %s: %s", chartPath, err)
	}
	if err := chartutil.SaveDir(p.Chart, chartPath); err != nil {
		return err
	}

	// save Dockerfile
	dockerfilePath := filepath.Join(dest, DockerfileName)
	exists, err := osutil.Exists(dockerfilePath)
	if err != nil {
		return err
	}
	if !exists {
		if err := ioutil.WriteFile(dockerfilePath, p.Dockerfile, 0644); err != nil {
			return err
		}
	}

	return nil
}
