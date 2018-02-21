package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/urfave/cli"
)

type MockedRunner struct {
	mock.Mock
	Runner
}

func (m *MockedRunner) Run(name string, arg ...string) error {
	args := m.Called(append([]string{name}, arg...))
	return args.Error(0)
}

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
	set.String("cluster", "cluster-0", "")
	err = checkParams(c)
	assert.NoError(t, err)
}

func TestGetProjectFromToken(t *testing.T) {
	token := "{\"project_id\":\"nyt-test-proj\"}"
	assert.Equal(t, "nyt-test-proj", getProjectFromToken(token))
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

func TestFetchCredentials(t *testing.T) {
	// Set cli.Context
	set := flag.NewFlagSet("test-set", 0)
	set.String("token", "{\"key\", \"val\"}", "")
	set.String("cluster", "cluster-0", "")
	set.String("zone", "us-east1", "")
	c := cli.NewContext(nil, set, nil)

	// No error
	testRunner := new(MockedRunner)
	testRunner.On("Run", []string{"gcloud", "auth", "activate-service-account", "--key-file", "/tmp/gcloud.json"}).Return(nil)
	testRunner.On("Run", []string{"gcloud", "container", "clusters", "get-credentials", "cluster-0", "--project", "nyt-test-proj", "--zone", "us-east1"}).Return(nil)
	err := fetchCredentials(c, "nyt-test-proj", testRunner)
	assert.NoError(t, err)
	testRunner.AssertExpectations(t)

	// Validate token file
	buf, err := ioutil.ReadFile("/tmp/gcloud.json")
	assert.Equal(t, "{\"key\", \"val\"}", string(buf))

	// Run() error
	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"gcloud", "auth", "activate-service-account", "--key-file", "/tmp/gcloud.json"}).Return(fmt.Errorf("e"))
	err = fetchCredentials(c, "nyt-test-proj", testRunner)
	assert.Error(t, err)
	testRunner.AssertExpectations(t)
}

func TestTemplateData(t *testing.T) {

}

func TestRenderTemplates(t *testing.T) {

}

func TestPrintKubectlVersion(t *testing.T) {

}

func TestSetNamespace(t *testing.T) {

}

func TestApplyManifests(t *testing.T) {

}

func TestWaitForRollout(t *testing.T) {

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
