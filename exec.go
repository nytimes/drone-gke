package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(name string, arg ...string) error
}

type BasicRunner struct {
	Runner
	dir    string
	env    []string
	stdout io.Writer
	stderr io.Writer
}

func NewBasicRunner(dir string, env []string, stdout, stderr io.Writer) *BasicRunner {
	return &BasicRunner{
		dir:    dir,
		env:    env,
		stdout: stdout,
		stderr: stderr,
	}
}

// Run executes the given program.
func (e *BasicRunner) Run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr

	// TODO: Extract this
	fmt.Println()
	fmt.Println("$", strings.Join(cmd.Args, " "))
	//--

	return cmd.Run()
}
