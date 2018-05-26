package drake

import (
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/Azure/draft/pkg/drake/dk"
	"github.com/Azure/draft/pkg/drake/parse"
)

// magicRebuildKey is used when hashing the output binary to ensure that we get
// a new binary even if nothing in the input files or generated mainfile has
// changed. This can be used when we change how we parse files, or otherwise
// change the inputs to the compiling process.
const (
	DrakefilePrefix = "Drakefile"
	magicRebuildKey = "v0.3"
)

var output = template.Must(template.New("").Funcs(map[string]interface{}{
	"lower": strings.ToLower,
	"lowerfirst": func(s string) string {
		r := []rune(s)
		return string(unicode.ToLower(r[0])) + string(r[1:])
	},
}).Parse(tpl))

const mainfile = "drake_output_file.go"

// set by ldflags when you "drake build"
var (
	commitHash string
	timestamp  string
	gitTag     = "v2"
)

//go:generate stringer -type=Command

// Command tracks invocations of drake that run without targets or other flags.
type Command int

const (
	None          Command = iota
	Version               // report the current version of drake
	Init                  // create a starting template for drake
	Clean                 // clean out old compiled drake binaries from the cache
	CompileStatic         // compile a static binary of the current directory
)

// Invocation contains the args for invoking a run of Drake.
type Invocation struct {
	Dir        string        // directory to read drakefiles from
	Force      bool          // forces recreation of the compiled binary
	Verbose    bool          // tells the drakefile to print out log statements
	List       bool          // tells the drakefile to print out a list of targets
	Help       bool          // tells the drakefile to print out help for a specific target
	Keep       bool          // tells drake to keep the generated main file after compiling
	Timeout    time.Duration // tells drake to set a timeout to running the targets
	CompileOut string        // tells drake to compile a static binary to this path, but not execute
	Stdout     io.Writer     // writer to write stdout messages to
	Stderr     io.Writer     // writer to write stderr messages to
	Stdin      io.Reader     // reader to read stdin from
	Args       []string      // args to pass to the compiled binary
}

// Parse parses the given args and returns structured data.  If parse returns
// flag.ErrHelp, the calling process should exit with code 0.
func Parse(stdout io.Writer, args []string) (inv Invocation, cmd Command, err error) {
	inv.Stdout = stdout
	fs := flag.FlagSet{}
	fs.SetOutput(stdout)
	fs.BoolVar(&inv.Force, "f", false, "force recreation of compiled drakefile")
	fs.BoolVar(&inv.Verbose, "v", false, "show verbose output when running drake targets")
	fs.BoolVar(&inv.List, "l", false, "list drake targets in this directory")
	fs.BoolVar(&inv.Help, "h", false, "show this help")
	fs.DurationVar(&inv.Timeout, "t", 0, "timeout in duration parsable format (e.g. 5m30s)")
	fs.BoolVar(&inv.Keep, "keep", false, "keep intermediate drake files around after running")
	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "show version info for the drake binary")
	var drakeInit bool
	fs.BoolVar(&drakeInit, "init", false, "create a starting template if no drake files exist")
	var clean bool
	fs.BoolVar(&clean, "clean", false, "clean out old generated binaries from CACHE_DIR")
	var compileOutPath string
	fs.StringVar(&compileOutPath, "compile", "", "path to which to output a static binary")

	fs.Usage = func() {
		fmt.Fprintln(stdout, "drake [options] [target]")
		fmt.Fprintln(stdout, "Options:")
		fs.PrintDefaults()
	}
	err = fs.Parse(args)
	if err == flag.ErrHelp {
		// parse will have already called fs.Usage()
		return inv, cmd, err
	}
	if err == nil && inv.Help && len(fs.Args()) == 0 {
		fs.Usage()
		// tell upstream, to just exit
		return inv, cmd, flag.ErrHelp
	}

	numFlags := 0
	switch {
	case drakeInit:
		numFlags++
		cmd = Init
	case compileOutPath != "":
		numFlags++
		cmd = CompileStatic
		inv.CompileOut = compileOutPath
		inv.Force = true
	case showVersion:
		numFlags++
		cmd = Version
	case clean:
		numFlags++
		cmd = Clean
		if fs.NArg() > 0 || fs.NFlag() > 1 {
			// Temporary dupe of below check until we refactor the other commands to use this check
			return inv, cmd, errors.New("-h, -init, -clean, -compile and -version cannot be used simultaneously")

		}
	}
	if inv.Help {
		numFlags++
	}

	// If verbose is still false, we're going to peek at the environment variable to see if
	// DRAKE_VERBOSE has been set. If so, we're going to use it for the value of DRAKE_VERBOSE.
	if inv.Verbose == false {
		envVerbose, err := strconv.ParseBool(os.Getenv("DRAKE_VERBOSE"))
		if err == nil {
			inv.Verbose = envVerbose
		}
	}

	if numFlags > 1 {
		return inv, cmd, errors.New("-h, -init, -clean, -compile and -version cannot be used simultaneously")
	}

	inv.Args = fs.Args()
	if inv.Help && len(inv.Args) > 1 {
		return inv, cmd, errors.New("-h can only show help for a single target")
	}

	return inv, cmd, err
}

// Invoke runs Drake with the given arguments.
func Invoke(inv Invocation) error {
	files, err := Drakefiles(inv.Dir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("No Drakefiles found in this directory")
	}

	exePath, err := ExeName(files)
	if err != nil {
		return err
	}
	if inv.CompileOut != "" {
		exePath = inv.CompileOut
	}

	if !inv.Force {
		if _, err := os.Stat(exePath); err == nil {
			return RunCompiled(inv, exePath)
		}
	}

	// parse wants dir + filenames... arg
	fnames := make([]string, 0, len(files))
	for i := range files {
		fnames = append(fnames, filepath.Base(files[i]))
	}

	info, err := parse.Package(inv.Dir, fnames)
	if err != nil {
		return err
	}

	hasDupes, names := CheckDupes(info)
	if hasDupes {
		return fmt.Errorf("Build targets must be case insensitive, thus the follow targets conflict: %v", names)
	}

	main := filepath.Join(inv.Dir, mainfile)
	if err := GenerateMainfile(main, info); err != nil {
		return err
	}
	if !inv.Keep {
		defer os.Remove(main)
	}
	files = append(files, main)
	if err := Compile(exePath, inv.Stdout, inv.Stderr, files); err != nil {
		return err
	}
	if !inv.Keep {
		// remove this file before we run the compiled version, in case the
		// compiled file screws things up. Yes, this doubles up with the above
		// defer, that's ok.
		os.Remove(main)
	}

	if inv.CompileOut != "" {
		return nil
	}

	return RunCompiled(inv, exePath)
}

// CheckDupes checks a package for duplicate target names.
func CheckDupes(info *parse.PkgInfo) (hasDupes bool, names map[string][]string) {
	names = map[string][]string{}
	lowers := map[string]bool{}
	for _, f := range info.Funcs {
		low := strings.ToLower(f.Name)
		if lowers[low] {
			hasDupes = true
		}
		lowers[low] = true
		names[low] = append(names[low], f.Name)
	}
	return hasDupes, names
}

type data struct {
	Funcs        []parse.Function
	DefaultError bool
	Default      string
	DefaultFunc  parse.Function
	Aliases      map[string]string
}

// Drakefiles returns the list of drakefiles in dir.
func Drakefiles(dir string) ([]string, error) {
	ctx := build.Default
	ctx.BuildTags = []string{"drake"}
	p, err := ctx.ImportDir(dir, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			return []string{}, nil
		}
		return nil, err
	}
	for i := range p.GoFiles {
		p.GoFiles[i] = filepath.Join(dir, p.GoFiles[i])
	}
	return p.GoFiles, nil
}

// Compile uses the go tool to compile the files into an executable at path.
func Compile(path string, stdout, stderr io.Writer, gofiles []string) error {
	c := exec.Command("go", append([]string{"build", "-o", path}, gofiles...)...)
	c.Env = os.Environ()
	c.Stderr = stderr
	c.Stdout = stdout
	err := c.Run()
	if err != nil {
		return errors.New("error compiling drakefiles")
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("failed to find compiled drakefile")
	}
	return nil
}

// GenerateMainfile creates the mainfile at path with the info from
func GenerateMainfile(path string, info *parse.PkgInfo) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("can't create mainfile: %v", err)
	}
	defer f.Close()

	data := data{
		Funcs:       info.Funcs,
		Default:     info.DefaultName,
		DefaultFunc: info.DefaultFunc,
		Aliases:     info.Aliases,
	}

	data.DefaultError = info.DefaultIsError

	if err := output.Execute(f, data); err != nil {
		return fmt.Errorf("can't execute mainfile template: %v", err)
	}
	return nil
}

// ExeName reports the executable filename that this version of Drake would
// create for the given drakefiles.
func ExeName(files []string) (string, error) {
	var hashes []string
	for _, s := range files {
		h, err := hashFile(s)
		if err != nil {
			return "", err
		}
		hashes = append(hashes, h)
	}
	// hash the mainfile template to ensure if it gets updated, we make a new
	// binary.
	hashes = append(hashes, fmt.Sprintf("%x", sha1.Sum([]byte(tpl))))
	sort.Strings(hashes)
	hash := sha1.Sum([]byte(strings.Join(hashes, "") + magicRebuildKey))
	filename := fmt.Sprintf("%x", hash)

	out := filepath.Join(dk.CacheDir(), filename)
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	return out, nil
}

func hashFile(fn string) (string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return "", fmt.Errorf("can't open input file: %v", err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("can't write data to hash: %v", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// RunCompiled runs an already-compiled drake command with the given args,
func RunCompiled(inv Invocation, exePath string) error {
	c := exec.Command(exePath, inv.Args...)
	c.Stderr = inv.Stderr
	c.Stdout = inv.Stdout
	c.Stdin = inv.Stdin
	c.Env = os.Environ()
	if inv.Verbose {
		c.Env = append(c.Env, "DRAKEFILE_VERBOSE=1")
	}
	if inv.List {
		c.Env = append(c.Env, "DRAKEFILE_LIST=1")
	}
	if inv.Help {
		c.Env = append(c.Env, "DRAKEFILE_HELP=1")
	}
	if inv.Timeout > 0 {
		c.Env = append(c.Env, fmt.Sprintf("DRAKEFILE_TIMEOUT=%s", inv.Timeout.String()))
	}
	return c.Run()
}