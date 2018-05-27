// +build drake

package main

import (
	"fmt"
	"log"

	"github.com/Azure/draft/pkg/drake/dk"
)

// This should work as a default - even if it's in a different file
var Default = ReturnsNilError

// this should not be a target because it returns a string
func ReturnsString() string {
	fmt.Println("more stuff")
	return ""
}

func TestVerbose() {
	log.Println("hi!")
}

func ReturnsVoid() {
	dk.Deps(f)
}

func f() {}
