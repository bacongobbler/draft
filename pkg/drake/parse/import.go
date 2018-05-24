package parse

import (
	"go/importer"
	"go/token"
	"go/types"
)

func getImporter(*token.FileSet) types.Importer {
	return importer.For("source", nil)
}
