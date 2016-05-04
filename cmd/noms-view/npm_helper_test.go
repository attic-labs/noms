package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteIfNecessary(t *testing.T) {
	assert := assert.New(t)

	tempdir, err := ioutil.TempDir("", "TestWriteIfNecessary")
	assert.NoError(err)
	defer os.RemoveAll(tempdir)

	tmpfile := func(name string) string {
		return path.Join(tempdir, name)
	}

	readfile := func(f *os.File) string {
		b := &bytes.Buffer{}
		io.Copy(b, f)
		return b.String()
	}

	var f *os.File
	f, err = os.Create(tmpfile("file"))
	assert.NoError(err)
	f.WriteString("my file")
	f.Close()

	err = os.Symlink(tmpfile("file"), tmpfile("symlink"))
	assert.NoError(err)

	npm := NpmHelper{tempdir, false}

	var did bool
	did, err = npm.writeIfNecessary("file", "my file #2")
	assert.False(did)
	assert.NoError(err)
	f, err = os.Open(tmpfile("file"))
	assert.NoError(err)
	assert.Equal("my file", readfile(f))
	f.Close()

	did, err = npm.writeIfNecessary("symlink", "my symlink")
	assert.False(did)
	assert.NoError(err)
	f, err = os.Open(tmpfile("symlink"))
	assert.NoError(err)
	assert.Equal("my file", readfile(f))
	f.Close()

	did, err = npm.writeIfNecessary("newfile", "my new file")
	assert.True(did)
	assert.NoError(err)
	f, err = os.Open(tmpfile("newfile"))
	assert.NoError(err)
	assert.Equal("my new file", readfile(f))
	f.Close()
}
