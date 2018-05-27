package main

import (
	"bytes"
	"testing"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

func TestPluginUpgradeCmd(t *testing.T) {
	// move this to e2e test suite soon
	target, err := newTestPluginEnv("", "")
	if err != nil {
		t.Fatal(err)
	}
	old, err := setupTestPluginEnv(target)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownTestPluginEnv(target, old)

	home := draftpath.Home(draftHome)
	buf := bytes.NewBuffer(nil)

	upgrade := &pluginUpgradeCmd{
		home: home,
		out:  buf,
	}

	if err := upgrade.run([]string{"server"}); err == nil {
		t.Errorf("expected plugin upgrade to err but did not")
	}

	install := &pluginInstallCmd{
		name: "env",
		home: home,
		out:  buf,
	}

	if err := install.run(); err != nil {
		t.Fatalf("Erroring installing plugin")
	}

	if err := upgrade.run([]string{"env"}); err != nil {
		t.Errorf("Erroring upgrading plugin: %v", err)
	}

}
