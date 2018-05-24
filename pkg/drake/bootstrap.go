//+build ignore

package main

import (
	"os"

	"github.com/Azure/draft/pkg/drake/drake"
)

// This is a bootstrap builder, to build drake when you don't already *have* drake.
// Run it like
// go run bootstrap.go
// and it will install drake with all the right flags created for you.

func main() {
	os.Args = []string{os.Args[0], "-v", "install"}
	os.Exit(drake.Main())
}
