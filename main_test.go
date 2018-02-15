package main

import (
	"bytes"
	"flag"
	// "io/ioutil"
	// "os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCheckParams(t *testing.T) {
	// Testing with cli.Context:
	// https://github.com/urfave/cli/blob/master/context_test.go#L10

	set := flag.NewFlagSet("test-set", 0)
	c := cli.NewContext(nil, set, nil)
	err := checkParams(c)
	assert.Error(t, err)

	// Complete
	set.String("token", "{}", "")
	set.String("zone", "us-east1", "")
	err = checkParams(c)
	assert.NoError(t, err)
}

func TestParseVars(t *testing.T) {
	set := flag.NewFlagSet("test-set", 0)
	c := cli.NewContext(nil, set, nil)
	vars, err := parseVars(c)
	assert.Equal(t, map[string]interface{}{}, vars)
	assert.NoError(t, err)

	// Invalid JSON
	set.String("vars", "{", "")
	vars, err = parseVars(c)
	assert.Equal(t, map[string]interface{}(nil), vars)
	assert.Error(t, err)

	set = flag.NewFlagSet("test-set", 0)
	set.String("vars", "{\"var0\": \"val0\", \"var1\": \"val1\"}", "")
	c = cli.NewContext(nil, set, nil)
	vars, err = parseVars(c)
	assert.Equal(t, map[string]interface{}{"var0": "val0", "var1": "val1"}, vars)
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
