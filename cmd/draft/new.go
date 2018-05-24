package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Azure/draft/pkg/draft"
	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/draft/manifest"
	"github.com/Azure/draft/pkg/osutil"
)

const (
	newDesc = `This command transforms the local directory to be deployable via 'draft up'.
`
)

type newCmd struct {
	name string
	out  io.Writer
	home draftpath.Home
}

func newNewCmd(out io.Writer) *cobra.Command {
	nc := &newCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "new <path>",
		Short: "create a new Draft application",
		Long:  newDesc,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			nc.normalizeApplicationName()
			return nc.complete(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nc.run()
		},
	}

	return cmd
}

func (c *newCmd) complete(args []string) error {
	if err := validateArgs(args, []string{"path"}); err != nil {
		return err
	}
	c.name = args[0]
	c.home = draftpath.Home(homePath())
	return nil
}

func (c *newCmd) run() error {
	mfest := manifest.New()
	mfest.Environments[manifest.DefaultEnvironmentName].Name = c.name

	if y, err := withinDraftDirectory(); y || err != nil {
		if err != nil {
			log.Debugln(err)
		}
		return fmt.Errorf("Can't initialize a new Draft application within the directory of another. Please change to a non-Draft directory first")
	}

	if err := draft.Create(c.name, c.name); err != nil {
		return fmt.Errorf("Failed initializing a new Draft application: %v", err)
	}

	fmt.Fprintln(c.out, "--> Ready to sail")
	return nil
}

func (c *newCmd) normalizeApplicationName() {
	if c.name == "" {
		return
	}

	nameIsUpperCase := false
	for _, char := range c.name {
		if unicode.IsUpper(char) {
			nameIsUpperCase = true
			break
		}
	}

	if !nameIsUpperCase {
		return
	}

	normalized := strings.ToLower(c.name)
	normalized = strings.Replace(normalized, "/", "-", -1)
	normalized = strings.Replace(normalized, "\\", "-", -1)
	fmt.Fprintf(
		c.out,
		"--> Application %s will be renamed to %s for docker compatibility\n",
		c.name,
		normalized,
	)
	c.name = normalized
}

func withinDraftDirectory() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	taskfileExists, err := osutil.Exists(filepath.Join(cwd, draft.DrakeFilename))
	if err != nil {
		return false, err
	}
	draftTomlExists, err := osutil.Exists(filepath.Join(cwd, draft.DraftTomlFilename))
	if err != nil {
		return false, err
	}
	return taskfileExists || draftTomlExists, nil
}
