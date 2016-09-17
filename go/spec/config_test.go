package spec

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/testify/assert"
)

const (
	ldbSpec    = "ldb:./local"
	memSpec    = "mem"
	httpSpec   = "http://test.com:8080/foo"
	remoteAlias = "origin"
)

var (
	ctestRoot = os.TempDir()

	ldbConfig = &Config{
		"",
		DefaultConfig{ ldbSpec },
		map[string]DbConfig{ remoteAlias: { httpSpec }},
	}

	httpConfig = &Config{
		"",
		DefaultConfig{ httpSpec },
		map[string]DbConfig{ remoteAlias: { ldbSpec }},
	}

	memConfig = &Config{
		"",
		DefaultConfig{ memSpec },
		map[string]DbConfig{ remoteAlias: { httpSpec }},
	}
)

type paths struct {
	home string
	config string
}

func getPaths(base string) paths {
	abs, err := filepath.Abs(ctestRoot)
	d.PanicIfError(err)
	abs, err = filepath.EvalSymlinks(ctestRoot)
	d.PanicIfError(err)
	home := filepath.Join(abs, base)
	config := filepath.Join(home, NomsConfigFile)
	return paths{ home, config }
}


func qualifyFilePath(path string) string {
	p, err := filepath.Abs(path)
	d.PanicIfError(err)
	return p
}

func assertDbSpecsEquiv(assert *assert.Assertions, expected string, actual string) {
	e, err := parseDatabaseSpec(expected)
	assert.NoError(err)
	if e.Protocol != "ldb" {
		assert.Equal(expected, actual)
	} else {
		a, err := parseDatabaseSpec(actual)
		assert.NoError(err)
		// remove leading . or ..
		ePath := strings.TrimPrefix(strings.TrimPrefix(e.Path, "."), ".")
		assert.True(e.Protocol == a.Protocol && strings.HasSuffix(a.Path, ePath),
			"expected suffix: " + ePath + "; actual path: " + actual,
		)
	}
}

func validateConfig(assert *assert.Assertions, file string, e *Config, a *Config) {
	assert.Equal(qualifyFilePath(file), qualifyFilePath(a.File))
	assertDbSpecsEquiv(assert, e.Default.Url, a.Default.Url)
	assert.Equal(len(e.Db), len(a.Db))
	for k, er := range e.Db {
		ar, ok := a.Db[k]
		assert.True(ok)
		assertDbSpecsEquiv(assert, er.Url, ar.Url)
	}
}

func writeConfig(assert *assert.Assertions, c *Config, home string) string {
	file, err := c.WriteTo(home)
	assert.NoError(err, home)
	return file
}


func TestConfig(t *testing.T) {
	assert := assert.New(t)
	path := getPaths("home")
	writeConfig(assert, ldbConfig, path.home)

	// Test from home
	assert.NoError(os.Chdir(path.home))
	c, err := FindNomsConfig()
	assert.NoError(err, path.config)
	validateConfig(assert, path.config, ldbConfig, c)

	// Test from subdir
	subdir := filepath.Join(path.home, "subdir")
	assert.NoError(os.MkdirAll(subdir, os.ModePerm))
	assert.NoError(os.Chdir(subdir))
	c, err = FindNomsConfig()
	assert.NoError(err, path.config)
	validateConfig(assert, path.config, ldbConfig, c)

	// Test from subdir with intervening .nomsconfig directory
	nomsDir := filepath.Join(subdir, NomsConfigFile)
	err = os.MkdirAll(nomsDir, os.ModePerm)
	assert.NoError(err, nomsDir)
	assert.NoError(os.Chdir(subdir))
	c, err = FindNomsConfig()
	assert.NoError(err, path.config)
	validateConfig(assert, path.config, ldbConfig, c)
}

func TestUnreadableConfig(t *testing.T) {
	assert := assert.New(t)
	path := getPaths("home.unreadable")
	writeConfig(assert, ldbConfig, path.home)
	assert.NoError(os.Chmod(path.config, 0333)) // write-only
	assert.NoError(os.Chdir(path.home))
	_, err := FindNomsConfig()
	assert.Error(err, path.config)
}

func TestNoConfig(t *testing.T) {
	assert := assert.New(t)
	path := getPaths("home.none")
	assert.NoError(os.MkdirAll(path.home, os.ModePerm))
	assert.NoError(os.Chdir(path.home))
	_, err := FindNomsConfig()
	assert.Equal(NoConfig, err)
}

func TestBadConfig(t *testing.T) {
	assert := assert.New(t)
	path := getPaths("home.bad")
	cfile := writeConfig(assert, ldbConfig, path.home)
	// overwrite with something invalid
	assert.NoError(ioutil.WriteFile(cfile, []byte("invalid config"), os.ModePerm))
	assert.NoError(os.Chdir(path.home))
	_, err := FindNomsConfig()
	assert.Error(err, path.config)
}

func TestQualifyingPaths(t *testing.T) {
	assert := assert.New(t)
	path := getPaths("home")
	assert.NoError(os.Chdir(path.home))

	for _, tc := range []*Config{ httpConfig, memConfig } {
		writeConfig(assert, tc, path.home)
		ac, err := FindNomsConfig()
		assert.NoError(err, path.config)
		validateConfig(assert, path.config, tc, ac)
	}
}

func TestCwd(t *testing.T) {
	assert := assert.New(t)
	cwd, err := os.Getwd()
	assert.NoError(err)
	cwd = filepath.Join(cwd, "test")
	abs, err := filepath.Abs("test")
	assert.NoError(err)

	assert.Equal(cwd, abs)
}


