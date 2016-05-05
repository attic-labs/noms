package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"syscall"

	"github.com/attic-labs/noms/d"
)

const (
	cmdPrefix = "noms-"
)

var (
	usePager = flag.Bool("use-pager", true, "page results when length exceeds terminal height")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "%s command [<command options>]\n", path.Base(os.Args[0]))
}

func main() {
	flag.Parse()

	if flag.NArg() == 0 || flag.Arg(0) == "help" {
		cmds := findCmds()
		if len(cmds) == 0 {
			fmt.Fprintf(os.Stderr, "Configuration error: unable to find any noms command in PATH.\n")
			os.Exit(-1)
		}

		fmt.Printf("Available commands:\n\n")
		for _, c := range cmds {
			fmt.Printf("%s\n", c)
		}
		return
	}

	cmdName := cmdPrefix + flag.Arg(0)
	executable, err := exec.LookPath(cmdName)
	if err != nil {
		d.Chk.Equal(err, exec.ErrNotFound)
		Usage()
		return
	}

	executeCmd(executable)
}

func findCmds() []string {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	cmds := []string{}
	prefixLen := len(cmdPrefix)
	for _, p := range paths {
		dir, err := os.Open(p)
		if err == nil {
			names, err := dir.Readdirnames(0)
			if err == nil {
				for _, n := range names {
					if strings.HasPrefix(n, cmdPrefix) && len(n) > prefixLen {
						fi, err := os.Stat(path.Join(p, n))
						d.Chk.NoError(err)
						if !fi.IsDir() && fi.Mode()&0111 != 0 {
							cmds = append(cmds, n[prefixLen:])
						}
					}
				}
			}
		}
	}
	sort.Strings(cmds)
	return cmds
}

func executeCmd(executable string) {
	lessCmd, err := exec.LookPath("less")
	if err != nil {
		*usePager = false
	}
	args := flag.Args()[1:]
	if len(args) == 0 {
		args = append(args, "-help")
	}
	c1 := exec.Command(executable, args...)
	c1.Stdin = os.Stdin
	c1.Stdout = os.Stdout
	c1.Stderr = os.Stderr

	if *usePager {
		c1.Stdout = nil
		c2 := exec.Command(lessCmd, []string{"-FSRX"}...)
		c2.Stdin, _ = c1.StdoutPipe()
		c2.Stdout = os.Stdout
		c2.Stderr = os.Stderr
		c2.Start()
		err = c1.Run()
		c2.Wait()
	} else {
		err = c1.Run()
	}

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
