package main

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

func TestParseConfig(t *testing.T) {
	testCases := []struct {
		configFile      string
		expectErr       bool
		pluginCount     int
		packRepoCount   int
		pluginRepoCount int
	}{
		{"", false, 0, 0, 0},
		{filepath.Join("testdata", "init", "configFile.toml"), false, 1, 2, 1},
		{filepath.Join("testdata", "init", "malformedConfigFile.toml"), true, 0, 0, 0},
		{filepath.Join("testdata", "init", "missingConfigFile.toml"), true, 0, 0, 0},
	}

	for _, tc := range testCases {
		resetEnvVars := unsetEnvVars()
		tempHome, teardown := tempDir(t, "draft-init")
		defer func() {
			teardown()
			resetEnvVars()
		}()

		cmd := &initCmd{
			home:       draftpath.Home(tempHome),
			out:        ioutil.Discard,
			configFile: tc.configFile,
		}

		conf, err := cmd.parseConfig()
		if err != nil && !tc.expectErr {
			t.Fatalf("Not expecting error but got error: %v", err)
		}
		if len(conf.Plugins) != tc.pluginCount {
			t.Errorf("Expected %v plugins, got %#v", tc.pluginCount, len(conf.Plugins))
		}
		if len(conf.PackRepositories) != tc.packRepoCount {
			t.Errorf("Expected %v pack repos, got %#v", tc.packRepoCount, len(conf.PackRepositories))
		}
		if len(conf.PluginRepositories) != tc.pluginRepoCount {
			t.Errorf("Expected %v plugin repos, got %#v", tc.pluginRepoCount, len(conf.PluginRepositories))
		}
	}
}
