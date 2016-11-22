package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Environ struct {
	dir    string
	env    []string
	stdout io.Writer
	stderr io.Writer
}

func NewEnviron(dir string, env []string, stdout, stderr io.Writer) *Environ {
	return &Environ{
		dir:    dir,
		env:    env,
		stdout: stdout,
		stderr: stderr,
	}
}

// Run executes the given program.
func (e *Environ) Run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr

	// TODO: Extract this
	fmt.Println("$", strings.Join(cmd.Args, " "))
	//--

	return cmd.Run()
}
