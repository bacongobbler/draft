package pack

import (
	"fmt"

	"github.com/otiai10/copy"
)

// CreateFrom scaffolds a directory with the src pack.
//
// This is different from calling
//
// pack, _ := pack.FromDir(src)
// pack.SaveDir(dest)
//
// because this function copies *everything* from src to dest.
func CreateFrom(dest, src string) error {
	// first do some validation that we are copying from a valid pack directory
	_, err := FromDir(src)
	if err != nil {
		return fmt.Errorf("could not load %s: %s", src, err)
	}

	return copy.Copy(src, dest)
}
