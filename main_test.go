package main

import (
    "bytes"
    // "io/ioutil"
    // "os"
    "testing"
    "strings"

    "github.com/stretchr/testify/assert"
    "github.com/urfave/cli"
)

func TestCheckParams(t *testing.T) {
    app := cli.NewApp()
    // c := NewContext(app,nil, nil)
    config, err := getConfig(c)
    assert.NoError(t, err)
}

func TestGetProjectFromToken(t *testing.T) {
    token := "{\"project_id\":\"nyt-test-proj\"}"
    assert.Equal(t, "nyt-test-proj", getProjectFromToken(token))
}

func TestApplyArgs(t *testing.T) {
    args := applyArgs(false, "/path/to/file/1")
    assert.Equal(t, []string{"apply", "--record", "--filename", "/path/to/file/1"}, args)

    args = applyArgs(true, "/path/to/file/2")
    assert.Equal(t, []string{"apply", "--record", "--dry-run", "--filename", "/path/to/file/2"}, args)
}

func TestPrintTrimmedError(t *testing.T) {
    output := &bytes.Buffer{}

    // Empty
    printTrimmedError(strings.NewReader(""), output)
    assert.Equal(t, "\n", output.String())

    // One line
    output.Reset()
    printTrimmedError(strings.NewReader("one line"), output)
    assert.Equal(t, "one line\n", output.String())

    // Mutiple lines
    output.Reset()
    printTrimmedError(strings.NewReader("line 1\nline 2\nline 3"), output)
    assert.Equal(t, "line 3\n", output.String())
}
