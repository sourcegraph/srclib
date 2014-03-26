package build

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/sourcegraph/rwvfs"
	"github.com/sqs/go-makefile/makefile"

	"sourcegraph.com/sourcegraph/client"
	"sourcegraph.com/sourcegraph/repo"
	"sourcegraph.com/sourcegraph/srcgraph/build/buildstore"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

var DataTypes = make(map[string]interface{})
var DataTypeNames = make(map[reflect.Type]string)

// RegisterDataType makes a build data type available by the provided name.
//
// When serializing build data of a certain type, the corresponding type name is
// included in the file basename. When deserializing build data from files, the
// file basename is parsed to determine the data type to deserialize into. For
// example, a data type with name "foo.v0" and type Foo{} would result in files
// named "whatever.foo.v0.json" being written, and those files would deserialize
// into Foo{} structs.
//
// The name should contain a version identifier so that types may be modified
// more easily; for example, name could be "graph.v0" (and a subsequent version
// could be registered as "graph.v1" with a different struct).
//
// The emptyInstance should be an empty instance of the type (usually a struct)
// that holds the build data; it is used to instantiate instances dynamically.
//
// If RegisterDataType is called twice with the same type or type name, if name
// is empty, or if emptyInstance is nil, it panics
func RegisterDataType(name string, emptyInstance interface{}) {
	if _, dup := DataTypes[name]; dup {
		panic("build: RegisterDataType called twice for type name " + name)
	}
	if emptyInstance == nil {
		panic("build: RegisterDataType emptyInstance is nil")
	}
	DataTypes[name] = emptyInstance

	typ := reflect.TypeOf(emptyInstance)
	if _, dup := DataTypeNames[typ]; dup {
		panic("build: RegisterDataType called twice for type " + typ.String())
	}
	if name == "" {
		panic("build: RegisterDataType name is nil")
	}
	DataTypeNames[typ] = name
	DataTypeNames[reflect.PtrTo(typ)] = name
}

var StorageDir = filepath.Join(os.TempDir(), "sg")

func init() {
	err := os.MkdirAll(StorageDir, 0700)
	if err != nil {
		log.Fatal(err)
	}
}

// localDataDir returns the directory in which to store build data for a
// specific commit in a repository.
func localDataDir(repoURI repo.URI, commitID string) string {
	return filepath.Join(StorageDir, string(repoURI), commitID)
}

var Storage = buildstore.WalkableRWVFS{rwvfs.OS(StorageDir)}

func dataTypeSuffix(typ reflect.Type) string {
	name, registered := DataTypeNames[typ]
	if !registered {
		panic("build: data type not registered: " + typ.String())
	}

	return name + ".json"
}

type DataFile interface {
	makefile.Target
	info() DataFileInfo
	filePath() string
}

type DataFileInfo struct {
	RepositoryURI repo.URI
	CommitID      string
	DataType      reflect.Type
}

type RepositoryCommitDataFile struct {
	DataFileInfo
}

func (f *RepositoryCommitDataFile) Name() string {
	return filepath.Join(localDataDir(f.RepositoryURI, f.CommitID), f.filePath())
}

func (f *RepositoryCommitDataFile) info() DataFileInfo { return f.DataFileInfo }
func (f *RepositoryCommitDataFile) filePath() string   { return dataTypeSuffix(f.DataType) }

type SourceUnitDataFile struct {
	DataFileInfo
	Unit unit.SourceUnit
}

func (f *SourceUnitDataFile) Name() string {
	return filepath.Join(localDataDir(f.RepositoryURI, f.CommitID), f.filePath())
}

func (f *SourceUnitDataFile) info() DataFileInfo { return f.DataFileInfo }

func (f *SourceUnitDataFile) filePath() string {
	return filepath.Clean(fmt.Sprintf("%s_%s", unit.MakeID(f.Unit), dataTypeSuffix(f.DataType)))
}

func DataFileSpec(f DataFile) client.BuildDataFileSpec {
	info := f.info()
	return client.BuildDataFileSpec{
		RepositorySpec: client.RepositorySpec{string(info.RepositoryURI)},
		CommitID:       info.CommitID,
		Path:           f.filePath(),
	}
}
