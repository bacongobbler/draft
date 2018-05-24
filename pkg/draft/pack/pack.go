package pack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/Azure/draft/pkg/osutil"
)

// Pack defines a Draft Starter Pack.
type Pack struct {
	// Chart is the Helm chart to be installed with the Pack.
	Chart *chart.Chart
	// Files are the files inside the Pack that will be installed.
	Files map[string]io.ReadCloser
}

// SaveDir saves a pack as files in a directory.
func (p *Pack) SaveDir(dest string) error {
	for relPath, f := range p.Files {
		path := filepath.Join(dest, relPath)
		exists, err := osutil.Exists(path)
		if err != nil {
			return err
		}
		if !exists {
			baseDir := filepath.Dir(path)
			if os.MkdirAll(baseDir, 0755) != nil {
				return fmt.Errorf("Error creating directory %v: %v", baseDir, err)
			}
			newfile, err := os.Create(path)
			if err != nil {
				return err
			}
			defer newfile.Close()
			defer f.Close()
			io.Copy(newfile, f)
		}
	}

	return nil
}
