package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	babelrc = `
{
  "env": {
    "production": {
      "presets": ["react", "es2015"],
      "plugins": [
        "syntax-async-functions",
        "syntax-flow",
        "transform-class-properties",
        "transform-regenerator",
        [
          "transform-runtime", {
            "polyfill": false,
            "regenerator": true
          }
        ]
      ]
    },
    "development": {
      "presets": ["react"],
      "plugins": [
        "syntax-async-functions",
        "syntax-flow",
        "transform-async-to-generator",
        "transform-class-properties",
        "transform-es2015-destructuring",
        "transform-es2015-modules-commonjs",
        "transform-es2015-parameters",
        [
          "transform-runtime", {
            "polyfill": false,
            "regenerator": true
          }
        ]
      ]
    },
    "es6": {
      "presets": ["react"],
      "plugins": [
        "syntax-async-functions",
        "syntax-flow",
        "transform-async-to-generator",
        "transform-class-properties",
        [
          "transform-runtime", {
            "polyfill": false,
            "regenerator": true
          }
        ]
      ]
    }
  }
}`

	// TODO: Make eslint optional.
	eslintrcJs = `
module.exports = require('@attic/eslintrc');
`

	// TODO: Make Flow optional.
	flowconfig = `
[ignore]
.*/node_modules/babel.*
.*/node_modules/babylon/.*
.*/node_modules/d3/.*
.*/node_modules/fbjs/.*
.*/node_modules/react/.*
.*/node_modules/y18n/.*

[options]
unsafe.enable_getters_and_setters=true
munge_underscores=true
suppress_comment=\\(.\\|\n\\)*\\$FlowIssue
`
)

var (
	srcFlag                 = flag.String("src", "index.js", "Source of main JavaScript file")
	outFlag                 = flag.String("out", "index.out.js", "Compiled JavaScript")
	packageJsonDependencies = []string{
		// TODO: Make eslint optional.
		"@attic/eslintrc",
		"@attic/noms",
		"@attic/webpack-config",
		"babel-cli",
		"babel-core",
		"babel-eslint",
		"babel-generator",
		"babel-plugin-syntax-async-functions",
		"babel-plugin-syntax-flow",
		"babel-plugin-transform-async-to-generator",
		"babel-plugin-transform-class-properties",
		"babel-plugin-transform-es2015-destructuring",
		"babel-plugin-transform-es2015-modules-commonjs",
		"babel-plugin-transform-es2015-parameters",
		"babel-plugin-transform-runtime",
		"babel-preset-es2015",
		"babel-preset-react",
		"babel-regenerator-runtime",
		"classnames",
		// TODO: Make Flow and React optional.
		"flow-bin",
		"react",
		"react-dom",
	}
)

type NpmHelper struct {
	dir     string
	verbose bool
}

func NewNpmHelper(dir string) *NpmHelper {
	return &NpmHelper{dir, true}
}

func (helper *NpmHelper) writeBabelrc() (bool, error) {
	return helper.writeIfNecessary(".babelrc", babelrc)
}

func (helper *NpmHelper) writeEslintrcJs() (bool, error) {
	return helper.writeIfNecessary(".eslintrc.js", eslintrcJs)
}

func (helper *NpmHelper) writeFlowconfig() (bool, error) {
	return helper.writeIfNecessary(".flowconfig", flowconfig)
}

func (helper *NpmHelper) writePackageJson() (bool, error) {
	type packageJsonType struct {
		Name            string            `json:"name"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
	}

	runWebpack := fmt.Sprintf("python node_modules/@attic/webpack-config/run.py --src %s --out %s", *srcFlag, *outFlag)
	pj := packageJsonType{
		"noms-view",
		// Dependencies are populated via "npm i -D ..." later.
		map[string]string{},
		map[string]string{
			"start": runWebpack + " development",
			"build": runWebpack + " production",
			// TODO: Make test script optional.
			"test": "eslint src/ && flow",
		},
	}

	did := false
	b, err := json.MarshalIndent(pj, "", "  ")

	if err == nil {
		// The Go json package escapes "&" to "\u0026", and doesn't write a newline at the end, fix it.
		// Mind you, "npm i -D ..." will probably fix that.
		str := strings.Replace(string(b), "\\u0026", "&", -1) + "\n"
		did, err = helper.writeIfNecessary("package.json", str)
	}

	return did, err
}

func (helper *NpmHelper) installPackageJsonDependencies() error {
	args := append([]string{"i", "-D"}, packageJsonDependencies...)
	cmd := exec.Command("npm", args...)
	cmd.Dir = helper.dir
	if helper.verbose {
		fmt.Println("Installing", len(packageJsonDependencies), "dependencies")
	}
	return cmd.Run()
}

func (helper *NpmHelper) writeIfNecessary(filename string, contents string) (bool, error) {
	filepath := path.Join(helper.dir, filename)

	if _, err := os.Lstat(filepath); err == nil {
		return false, nil
	}

	f, err := os.Create(filepath)
	if err == nil {
		defer f.Close()
		if helper.verbose {
			fmt.Println("Writing", filename)
		}
		f.WriteString(contents)
	}

	return err == nil, err
}

func (helper *NpmHelper) Install() error {
	_, err := helper.writeBabelrc()

	if err == nil {
		_, err = helper.writeEslintrcJs()
	}

	if err == nil {
		_, err = helper.writeFlowconfig()
	}

	if err == nil {
		var did bool
		if did, err = helper.writePackageJson(); did {
			err = helper.installPackageJsonDependencies()
		}
	}

	return err
}

func (helper *NpmHelper) Start() error {
	cmd := exec.Command("npm", "start")
	cmd.Dir = helper.dir
	return cmd.Run()
}
