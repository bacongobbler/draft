package dk

import (
	"os"

	"github.com/Azure/draft/pkg/draft/draftpath"
)

// CacheEnv is the environment variable that users may set to change the
// location where drake stores its compiled binaries.
const CacheEnv = "DRAKEFILE_CACHE"

// verboseEnv is the environment variable that indicates the user requested
// verbose mode when running a drakefile.
const verboseEnv = "DRAKEFILE_VERBOSE"

// Verbose reports whether a drakefile was run with the verbose flag.
func Verbose() bool {
	return os.Getenv(verboseEnv) != ""
}

// CacheDir returns the directory where drake caches compiled binaries.  It
// defaults to $DRAFT_HOME/drake, but may be overridden by the DRAKEFILE_CACHE
// environment variable.
func CacheDir() string {
	d := os.Getenv(CacheEnv)
	if d != "" {
		return d
	}
	return draftpath.Home(draftpath.DefaultHome()).Path("drake")
}
