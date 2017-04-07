// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/testify/assert"
)

func makeFileManifestTempDir(t *testing.T) *fileManifest {
	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	return &fileManifest{dir: dir}
}

func TestFileManifestLoadIfExists(t *testing.T) {
	assert := assert.New(t)
	fm := makeFileManifestTempDir(t)
	defer os.RemoveAll(fm.dir)

	exists, vers, root, tableSpecs := fm.LoadIfExists(nil)
	assert.False(exists)

	// Simulate another process writing a manifest (with an old Noms version).
	lock := computeAddr([]byte("locker"))
	newRoot := hash.Of([]byte("new root"))
	tableName := hash.Of([]byte("table1"))
	err := clobberManifest(fm.dir, strings.Join([]string{StorageVersion, "0", lock.String(), newRoot.String(), tableName.String(), "0"}, ":"))
	assert.NoError(err)

	// LoadIfExists should now reflect the manifest written above.
	exists, vers, root, tableSpecs = fm.LoadIfExists(nil)
	assert.True(exists)
	assert.Equal("0", vers)
	assert.Equal(newRoot, root)
	if assert.Len(tableSpecs, 1) {
		assert.Equal(tableName.String(), tableSpecs[0].name.String())
		assert.Equal(uint32(0), tableSpecs[0].chunkCount)
	}
}

func TestFileManifestLoadIfExistsHoldsLock(t *testing.T) {
	assert := assert.New(t)
	fm := makeFileManifestTempDir(t)
	defer os.RemoveAll(fm.dir)

	// Simulate another process writing a manifest.
	lock := computeAddr([]byte("locker"))
	newRoot := hash.Of([]byte("new root"))
	tableName := hash.Of([]byte("table1"))
	err := clobberManifest(fm.dir, strings.Join([]string{StorageVersion, constants.NomsVersion, lock.String(), newRoot.String(), tableName.String(), "0"}, ":"))
	assert.NoError(err)

	// LoadIfExists should now reflect the manifest written above.
	exists, vers, root, tableSpecs := fm.LoadIfExists(func() {
		// This should fail to get the lock, and therefore _not_ clobber the manifest.
		lock := computeAddr([]byte("newlock"))
		badRoot := hash.Of([]byte("bad root"))
		b, err := tryClobberManifest(fm.dir, strings.Join([]string{StorageVersion, "0", lock.String(), badRoot.String(), tableName.String(), "0"}, ":"))
		assert.NoError(err, string(b))
	})

	assert.True(exists)
	assert.Equal(constants.NomsVersion, vers)
	assert.Equal(newRoot, root)
	if assert.Len(tableSpecs, 1) {
		assert.Equal(tableName.String(), tableSpecs[0].name.String())
		assert.Equal(uint32(0), tableSpecs[0].chunkCount)
	}
}

func TestFileManifestUpdateWontClobberOldVersion(t *testing.T) {
	assert := assert.New(t)
	fm := makeFileManifestTempDir(t)
	defer os.RemoveAll(fm.dir)

	// Simulate another process having already put old Noms data in dir/.
	err := clobberManifest(fm.dir, strings.Join([]string{StorageVersion, "0", addr{}.String(), hash.Hash{}.String()}, ":"))
	assert.NoError(err)

	assert.Panics(func() { fm.Update(nil, hash.Hash{}, hash.Hash{}, nil) })
}

func TestFileManifestUpdateEmpty(t *testing.T) {
	assert := assert.New(t)
	fm := makeFileManifestTempDir(t)
	defer os.RemoveAll(fm.dir)

	actual, tableSpecs, err := fm.Update(nil, hash.Hash{}, hash.Hash{}, nil)
	assert.NoError(err)
	assert.True(actual.IsEmpty())
	assert.Empty(tableSpecs)

	fm2 := newFileManifest(fm.dir) // Open existent, but empty manifest
	exists, _, root, tableSpecs := fm2.LoadIfExists(nil)
	assert.True(exists)
	assert.True(root.IsEmpty())
	assert.Empty(tableSpecs)

	actual, tableSpecs, err = fm2.Update(nil, hash.Hash{}, hash.Hash{}, nil)
	assert.NoError(err)
	assert.True(actual.IsEmpty())
	assert.Empty(tableSpecs)
}

func TestFileManifestUpdate(t *testing.T) {
	assert := assert.New(t)
	fm := makeFileManifestTempDir(t)
	defer os.RemoveAll(fm.dir)

	// First, test winning the race against another process.
	newRoot := hash.Of([]byte("new root"))
	specs := []tableSpec{{computeAddr([]byte("a")), 3}}
	actual, tableSpecs, err := fm.Update(specs, hash.Hash{}, newRoot, func() {
		// This should fail to get the lock, and therefore _not_ clobber the manifest. So the Update should succeed.
		lock := computeAddr([]byte("locker"))
		newRoot2 := hash.Of([]byte("new root 2"))
		b, err := tryClobberManifest(fm.dir, strings.Join([]string{StorageVersion, constants.NomsVersion, lock.String(), newRoot2.String()}, ":"))
		assert.NoError(err, string(b))
	})
	assert.NoError(err)
	assert.Equal(newRoot, actual)
	assert.Equal(specs, tableSpecs)

	// Now, test the case where the optimistic lock fails, and someone else updated the root since last we checked.
	newRoot2 := hash.Of([]byte("new root 2"))
	actual, tableSpecs, err = fm.Update(nil, hash.Hash{}, newRoot2, nil)
	assert.IsType(err, errOptimisticLockFailedRoot)
	assert.Equal(newRoot, actual)
	assert.Equal(specs, tableSpecs)
	actual, tableSpecs, err = fm.Update(nil, actual, newRoot2, nil)
	assert.NoError(err)
	assert.Equal(newRoot2, actual)
	assert.Empty(tableSpecs)

	// Now, test the case where the optimistic lock fails because someone else updated only the tables since last we checked
	lock := computeAddr([]byte("locker"))
	tableName := computeAddr([]byte("table1"))
	err = clobberManifest(fm.dir, strings.Join([]string{StorageVersion, constants.NomsVersion, lock.String(), newRoot2.String(), tableName.String(), "1"}, ":"))
	assert.NoError(err)

	newRoot3 := hash.Of([]byte("new root 3"))
	actual, tableSpecs, err = fm.Update(nil, newRoot3, newRoot2, nil)
	assert.IsType(err, errOptimisticLockFailedTables)
	assert.Equal(newRoot2, actual)
	assert.Equal([]tableSpec{{tableName, 1}}, tableSpecs)
}

// tryClobberManifest simulates another process trying to access dir/manifestFileName concurrently. To avoid deadlock, it does a non-blocking lock of dir/lockFileName. If it can get the lock, it clobbers the manifest.
func tryClobberManifest(dir, contents string) ([]byte, error) {
	return runClobber(dir, contents)
}

// clobberManifest simulates another process writing dir/manifestFileName concurrently. It ignores the lock file, so it's up to the caller to ensure correctness.
func clobberManifest(dir, contents string) error {
	if err := ioutil.WriteFile(filepath.Join(dir, lockFileName), nil, 0666); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dir, manifestFileName), []byte(contents), 0666)
}

func runClobber(dir, contents string) ([]byte, error) {
	_, filename, _, _ := runtime.Caller(1)
	clobber := filepath.Join(filepath.Dir(filename), "test/manifest_clobber.go")
	mkPath := func(f string) string {
		return filepath.Join(dir, f)
	}

	c := exec.Command("go", "run", clobber, mkPath(lockFileName), mkPath(manifestFileName), contents)
	return c.CombinedOutput()
}
