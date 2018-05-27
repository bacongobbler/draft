package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"
	"github.com/Azure/draft/pkg/testing/helpers"
)

func TestPluginRemoveCmd(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	target, err := newTestPluginEnv("", "")
	if err != nil {
		t.Fatal(err)
	}
	old, err := setupTestPluginEnv(target)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTestPluginEnv(target, old)

	remove := &pluginUninstallCmd{
		home:  draftpath.Home(homePath()),
		out:   buf,
		names: []string{"echo"},
	}

	helpers.CopyTree(t, filepath.Join("testdata", "plugins"), plugin.Home(remove.home.Plugins()).Installed())

	if err := remove.run(); err != nil {
		t.Errorf("Error removing plugin: %v", err)
	}

	expectedOutput := "echo: uninstalled in"
	actual := buf.String()

	if !strings.Contains(actual, expectedOutput) {
		t.Errorf("Expected '%v', got '%v'", expectedOutput, actual)
	}
}
