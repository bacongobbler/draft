package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/draft/pkg/draft/draftpath"
	"github.com/Azure/draft/pkg/drake/dk"
	"github.com/Azure/draft/pkg/drake/drake"
	"github.com/spf13/cobra"
)

type taskCmd struct {
	stdout         io.Writer
	stdin          io.Reader
	stderr         io.Writer
	home           draftpath.Home
	force          bool
	verbose        bool
	list           bool
	timeout        time.Duration
	keep           bool
	clean          bool
	compileOutPath string
}

func newTaskCmd(stdout io.Writer, stdin io.Reader, stderr io.Writer) *cobra.Command {
	t := taskCmd{
		stdout: stdout,
		stdin:  stdin,
		stderr: stderr,
	}

	cmd := &cobra.Command{
		Use:   "task <target>",
		Short: "Run Draft tasks defined in a Drakefile",
		RunE: func(cmd *cobra.Command, args []string) error {
			t.home = draftpath.Home(defaultDraftHome())
			return t.run(args)
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&t.clean, "clean", false, "clean out old generated binaries from CACHE_DIR")
	fs.StringVar(&t.compileOutPath, "compile", "", "path to which to output a static binary")
	fs.BoolVar(&t.force, "force", false, "force recreation of compiled drakefile")
	fs.BoolVar(&t.keep, "keep", false, "keep intermediate drake files around after running")
	fs.BoolVar(&t.list, "list", false, "list drake targets in this directory")
	fs.DurationVar(&t.timeout, "timeout", 0, "timeout in duration parsable format (e.g. 5m30s)")
	fs.BoolVar(&t.verbose, "verbose", false, "show verbose output when running drake targets")

	return cmd
}

func (t *taskCmd) run(args []string) error {
	inv := drake.Invocation{
		Dir:    ".",
		Stdout: t.stdout,
		Stdin:  t.stdin,
		Stderr: t.stderr,
	}

	switch {
	case t.clean:
		dir := dk.CacheDir()
		if err := removeContents(dir); err != nil {
			return err
		}
		fmt.Fprintln(t.stdout, dir, "cleaned")
		return nil
	case t.compileOutPath != "":
		inv.CompileOut = t.compileOutPath
		inv.Force = true
	case t.force:
		inv.Force = true
	case t.keep:
		inv.Keep = true
	case t.list:
		inv.List = true
	case t.timeout > 0:
		inv.Timeout = t.timeout
	case t.verbose:
		inv.Verbose = true
	}

	inv.Args = args
	return drake.Invoke(inv)
}

func removeContents(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		err = os.Remove(filepath.Join(dir, f.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}
