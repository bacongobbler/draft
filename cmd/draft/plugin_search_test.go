package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/plugin"
)

func TestPluginSearchCmd(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	resetEnvVars := unsetEnvVars()
	defer resetEnvVars()

	searchCmd := &pluginSearchCmd{
		home: plugin.Home(filepath.Join("testdata", "drafthome", "plugins")),
		out:  buf,
	}

	searchCmd.run([]string{})

	expectedOutput := "NAME   \tREPOSITORY\tVERSION\tDESCRIPTION      \nargs   \tfoo       \t0.1.0  \tThis echos args  \necho   \tfoo       \t0.1.0  \tThis echos stuff \nfullenv\tfoo       \t0.1.0  \tshow all env vars\nhome   \tfoo       \t0.1.0  \tshow DRAFT_HOME  \n"

	actual := buf.String()
	if strings.Compare(actual, expectedOutput) != 0 {
		t.Errorf("Expected %q, Got %q", expectedOutput, actual)
	}
}

func TestEmptyResultsOnPluginListCmd(t *testing.T) {
	target, err := newTestPluginEnv("", "")
	if err != nil {
		t.Fatal(err)
	}

	old, err := setupTestPluginEnv(target)
	if err != nil {
		t.Fatal(err)
	}

	defer teardownTestPluginEnv(target, old)

	buf := bytes.NewBuffer(nil)
	list := &pluginListCmd{
		home: draftpath.Home(homePath()),
		out:  buf,
	}

	if err := list.run([]string{}); err != nil {
		t.Errorf("draft plugin list error: %v", err)
	}

	expectedOutput := "No plugins found\n"
	actual := buf.String()
	if strings.Compare(actual, expectedOutput) != 0 {
		t.Errorf("Expected %s, got %s", expectedOutput, actual)
	}

}
