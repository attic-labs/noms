package spec

import (
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/spec/lite"
	"github.com/attic-labs/noms/go/types"

	"github.com/attic-labs/noms/go/spec/aws"
	_ "github.com/attic-labs/noms/go/spec/http"
	_ "github.com/attic-labs/noms/go/spec/nbs"
)

type (
	Spec         = spec.Spec
	SpecOptions  = spec.SpecOptions
	AbsolutePath = spec.AbsolutePath
)

var (
	ExternalProtocols = spec.ExternalProtocols
	GetAWSSession     = aws.GetAWSSession
)

const (
	Separator            = spec.Separator
	CommitMetaDateFormat = spec.CommitMetaDateFormat
)

// ForDatabase parses a spec for a Database.
func ForDatabase(sp string) (Spec, error) {
	return spec.ForDatabase(sp)
}

// ForPath parses a spec for a path to a Value.
func ForPath(sp string) (Spec, error) {
	return spec.ForPathOpts(sp, SpecOptions{})
}

// ForDatabaseOpts parses a spec for a Database.
func ForDatabaseOpts(sp string, opts SpecOptions) (Spec, error) {
	return spec.ForDatabaseOpts(sp, opts)
}

// ForDataset parses a spec for a Dataset.
func ForDataset(sp string) (Spec, error) {
	return spec.ForDataset(sp)
}

func NewAbsolutePath(str string) (AbsolutePath, error) {
	return spec.NewAbsolutePath(str)
}

func ReadAbsolutePaths(db datas.Database, paths ...string) ([]types.Value, error) {
	return spec.ReadAbsolutePaths(db, paths...)
}

func CreateDatabaseSpecString(protocol, db string) string {
	return spec.CreateDatabaseSpecString(protocol, db)
}

func CreateValueSpecString(protocol, db, path string) string {
	return spec.CreateValueSpecString(protocol, db, path)
}

func CreateHashSpecString(protocol, db string, h hash.Hash) string {
	return spec.CreateHashSpecString(protocol, db, h)
}

func CreateCommitMetaStruct(db datas.Database, date, message string, keyValueStrings map[string]string, keyValuePaths map[string]types.Value) (types.Struct, error) {
	return spec.CreateCommitMetaStruct(db, date, message, keyValueStrings, keyValuePaths)
}
