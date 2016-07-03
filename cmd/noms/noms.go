// Copyright 2016 The Noms Authors. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
)

const (
	cmdPrefix = "noms-"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s command [command-args]\n\n", path.Base(os.Args[0]))
	if hasDefinedFlags(flag.CommandLine) {
		fmt.Fprintf(os.Stderr, "Flags:\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}
	fmt.Fprintf(os.Stderr, "Commands:\n\n")
	fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(listCmds(), "\n  "))
	fmt.Fprintf(os.Stderr, "\nSee noms <command> -h for information on each available command.\n\n")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 || flag.Arg(0) == "help" {
		usage()
		os.Exit(1)
	}

	cmd := findCmd(flag.Arg(0))
	if cmd == "" {
		fmt.Fprintf(os.Stderr, "error: %s is not an available command\n", flag.Arg(0))
		usage()
		os.Exit(1)
	}

	executeCmd(cmd)
}

func hasDefinedFlags(fs *flag.FlagSet) (hasFlags bool) {
	fs.VisitAll(func(*flag.Flag) {
		hasFlags = true
	})
	return
}

func findCmd(name string) (cmd string) {
	nomsName := cmdPrefix + name
	if runtime.GOOS == "windows" {
		nomsName += ".exe" // ugh...
	}
	forEachDir(func(dir *os.File) (stop bool) {
		if isNomsExecutable(dir, nomsName) {
			cmd = path.Join(dir.Name(), nomsName)
			stop = true
		}
		return
	})
	return
}

func stripFileExtension(cmd string) string {
	return strings.TrimSuffix(cmd, filepath.Ext(cmd))
}

func listCmds() []string {
	cmds := []string{}

	encountered := map[string]bool{}
	forEachDir(func(dir *os.File) (stop bool) {
		// dir.Readdirnames may return an error, but |names| may still contain valid files.
		names, _ := dir.Readdirnames(0)
		for _, n := range names {
			if isNomsExecutable(dir, n) {
				cmd := stripFileExtension(n[len(cmdPrefix):])
				if (!encountered[cmd]) {
					cmds = append(cmds, cmd)
					encountered[cmd] = true
				}
				
			}
		}
		return
	})

	sort.Strings(cmds)
	return cmds
}

func forEachDir(cb func(dir *os.File) bool) {
	lookups := []struct {
		Env    string
		Suffix string
	}{
		{"PATH", ""},
		{"GOPATH", "bin"},
	}

	seen := map[string]bool{}

	for _, lookup := range lookups {
		env := os.Getenv(lookup.Env)
		if env == "" {
			continue
		}

		paths := strings.Split(env, string(os.PathListSeparator))
		for _, p := range paths {
			p := path.Join(p, lookup.Suffix)

			if seen[p] {
				continue
			}

			seen[p] = true

			if dir, err := os.Open(p); err == nil && cb(dir) {
				return
			}
		}
	}
}

func executeCmd(executable string) {
	args := flag.Args()[1:]
	if len(args) == 0 {
		args = append(args, "-help")
	}
	nomsCmd := exec.Command(executable, args...)
	nomsCmd.Stdin = os.Stdin
	nomsCmd.Stdout = os.Stdout
	nomsCmd.Stderr = os.Stderr

	err := nomsCmd.Run()
	if err != nil {
		switch t := err.(type) {
		case *exec.ExitError:
			status := t.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
			os.Exit(status)
		default:
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(-1)
		}
	}
}

func isNomsExecutable(dir *os.File, name string) bool {
	if !strings.HasPrefix(name, cmdPrefix) || len(name) == len(cmdPrefix) {
		return false
	}

	fi, err := os.Stat(path.Join(dir.Name(), name))
	return err == nil && !fi.IsDir()
}
