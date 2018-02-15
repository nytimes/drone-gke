package main

import (
	"bytes"
	"flag"
	// "io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestCheckParams(t *testing.T) {
	// Testing with cli.Context:
	// https://github.com/urfave/cli/blob/master/context_test.go#L10

	// No args set
	set := flag.NewFlagSet("test-set", 0)
	c := cli.NewContext(nil, set, nil)
	err := checkParams(c)
	assert.Error(t, err)

	// Required args set
	set.String("token", "{}", "")
	set.String("zone", "us-east1", "")
	err = checkParams(c)
	assert.NoError(t, err)
}

func TestParseVars(t *testing.T) {
	// No vars set
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

	// Valid
	set = flag.NewFlagSet("test-set", 0)
	set.String("vars", "{\"var0\": \"val0\", \"var1\": \"val1\"}", "")
	c = cli.NewContext(nil, set, nil)
	vars, err = parseVars(c)
	assert.Equal(t, map[string]interface{}{"var0": "val0", "var1": "val1"}, vars)
	assert.NoError(t, err)
}

func TestParseSecrets(t *testing.T) {
	// Unset all secrets first
	os.Clearenv()

	// No secrets
	secrets, err := parseSecrets()
	assert.Equal(t, map[string]string{}, secrets)
	assert.NoError(t, err)

	// Normal
	os.Setenv("SECRET_TEST0", "test0")
	os.Setenv("SECRET_TEST1", "test1")
	os.Setenv("SECRET_BASE64_TEST0", "dGVzdDA=")
	secrets, err = parseSecrets()
	assert.Equal(
		t,
		map[string]string{
			"SECRET_TEST0":        "dGVzdDA=",
			"SECRET_TEST1":        "dGVzdDE=",
			"SECRET_BASE64_TEST0": "dGVzdDA=",
		},
		secrets)

	assert.NoError(t, err)

	// Empty string is not allowed
	os.Clearenv()
	os.Setenv("SECRET_TEST", "")
	secrets, err = parseSecrets()
	assert.Equal(t, map[string]string(nil), secrets)
	assert.Error(t, err)

	// Not able to use os.Setenv() to set env vars without "=", or duplicate keys
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
