package builder

import (
	"path/filepath"
	"testing"

	"github.com/Azure/draft/pkg/draft/manifest"
)

func TestArchiveSrc(t *testing.T) {
	ctx := &Context{
		AppDir: filepath.Join("testdata", "simple"),
		Env: &manifest.Environment{
			Dockerfile: "",
		},
	}

	if err := archiveSrc(ctx); err != nil {
		t.Error(err)
	}

	if len(ctx.Archive) == 0 {
		t.Errorf("expected non-zero archive length, got %d", len(ctx.Archive))
	}
}
