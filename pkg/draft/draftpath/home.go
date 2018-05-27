package draftpath

import (
	"os"
	"path/filepath"
	"runtime"
)

const HomeEnvVar = "DRAFT_HOME"
const PluginEnvVar = `DRAFT_PLUGIN`

// Home describes the location of a CLI configuration.
//
// This helper builds paths relative to a Draft Home directory.
type Home string

// Path returns Home with elements appended.
func (h Home) Path(elem ...string) string {
	p := []string{h.String()}
	p = append(p, elem...)
	return filepath.Join(p...)
}

// Config returns the path to the Draft config file.
func (h Home) Config() string {
	return h.Path("config.toml")
}

// Packs returns the path to the Draft starter packs.
func (h Home) Packs() string {
	return h.Path("packs")
}

// Logs returns the path to the Draft logs.
func (h Home) Logs() string {
	return h.Path("logs")
}

// Plugins returns the path to the Draft plugins.
func (h Home) Plugins() string {
	plugdirs := os.Getenv(PluginEnvVar)

	if plugdirs == "" {
		plugdirs = h.Path("plugins")
	}

	return plugdirs
}

// String returns Home as a string.
//
// Implements fmt.Stringer.
func (h Home) String() string {
	return string(h)
}

// DefaultHome gives the default value for $(draft home)
func DefaultHome() string {
	if home := os.Getenv(HomeEnvVar); home != "" {
		return home
	}

	homeEnvPath := os.Getenv("HOME")
	if homeEnvPath == "" && runtime.GOOS == "windows" {
		homeEnvPath = os.Getenv("USERPROFILE")
	}

	return filepath.Join(homeEnvPath, ".draft")
}
