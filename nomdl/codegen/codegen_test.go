package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/datas"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/nomdl/codegen/code"
	"github.com/attic-labs/noms/nomdl/pkg"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
	"github.com/stretchr/testify/assert"
)

func assertOutput(inPath, goldenPath string, t *testing.T) {
	assert := assert.New(t)
	emptyDS := datas.NewDataStore(chunks.NewMemoryStore()) // Will be DataStore containing imports

	depsDir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	defer os.RemoveAll(depsDir)

	inFile, err := os.Open(inPath)
	assert.NoError(err)
	defer inFile.Close()

	goldenFile, err := os.Open(goldenPath)
	assert.NoError(err)
	defer goldenFile.Close()
	goldenBytes, err := ioutil.ReadAll(goldenFile)
	d.Chk.NoError(err)

	var buf bytes.Buffer
	pkg := pkg.ParseNomDL("gen", inFile, filepath.Dir(inPath), emptyDS)
	written := map[string]bool{}
	gen := newCodeGen(&buf, getBareFileName(inPath), written, depsMap{}, pkg)
	gen.WritePackage()

	bs := buf.Bytes()
	assert.Equal(string(goldenBytes), string(bs), "%s did not generate the same string", inPath)
}

func TestGeneratedFiles(t *testing.T) {
	files, err := filepath.Glob("test/*.noms")
	d.Chk.NoError(err)
	assert.NotEmpty(t, files)
	for _, n := range files {
		_, file := filepath.Split(n)
		if file == "struct_with_imports.noms" {
			// We are not writing deps in this test so lookup by ref does not work.
			continue
		}
		if file == "struct_with_list.noms" || file == "struct_with_dup_list.noms" {
			// These two files race to write ListOfUint8
			continue
		}
		assertOutput(n, filepath.Join("test", "gen", file+".js"), t)
	}
}

func TestSkipDuplicateTypes(t *testing.T) {
	assert := assert.New(t)
	dir, err := ioutil.TempDir("", "codegen_test_")
	assert.NoError(err)
	defer os.RemoveAll(dir)

	leaf1 := types.NewPackage([]*types.Type{
		types.MakeStructType("S1", []types.Field{
			types.Field{"f", types.MakeListType(types.NumberType), false},
			types.Field{"e", types.MakeType(ref.Ref{}, 0), false},
		}, []types.Field{}),
	}, []ref.Ref{})
	leaf2 := types.NewPackage([]*types.Type{
		types.MakeStructType("S2", []types.Field{
			types.Field{"f", types.MakeListType(types.NumberType), false},
		}, []types.Field{}),
	}, []ref.Ref{})

	written := map[string]bool{}
	tag1 := code.ToTag(leaf1.Ref())
	leaf1Path := filepath.Join(dir, tag1+".js")
	generateAndEmit(tag1, leaf1Path, written, depsMap{}, pkg.Parsed{Package: leaf1, Name: "p"})

	tag2 := code.ToTag(leaf2.Ref())
	leaf2Path := filepath.Join(dir, tag2+".js")
	generateAndEmit(tag2, leaf2Path, written, depsMap{}, pkg.Parsed{Package: leaf2, Name: "p"})

	code, err := ioutil.ReadFile(leaf2Path)
	assert.NoError(err)
	assert.NotContains(string(code), "type ListOfNumber")
}

func TestCommitNewPackages(t *testing.T) {
	assert := assert.New(t)
	ds := datas.NewDataStore(chunks.NewMemoryStore())
	pkgDS := dataset.NewDataset(ds, "packages")

	dir, err := ioutil.TempDir("", "")
	assert.NoError(err)
	defer os.RemoveAll(dir)
	inFile := filepath.Join(dir, "in.noms")
	err = ioutil.WriteFile(inFile, []byte("struct Simple{a:Bool}"), 0600)
	assert.NoError(err)

	p := parsePackageFile("name", inFile, pkgDS)
	localPkgs := refSet{p.Ref(): true}
	pkgDS = generate("name", inFile, filepath.Join(dir, "out.js"), dir, map[string]bool{}, p, localPkgs, pkgDS)
	s := pkgDS.Head().Get(datas.ValueField).(types.Set)
	assert.EqualValues(1, s.Len())
	tr := s.First().(types.Ref).TargetValue(ds).(types.Package).Types()[0]
	assert.EqualValues(types.StructKind, tr.Kind())
}
