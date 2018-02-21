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
	// https://godoc.org/github.com/stretchr/testify/mock
	// Arguments given in .On()
	args := m.Called(append([]string{name}, arg...))
	// Returns error given in .Return()
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
	token := "{\"project_id\":\"test-project\"}"
	assert.Equal(t, "test-project", getProjectFromToken(token))
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

	// No error
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

	// No error
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
	testRunner.On("Run", []string{"gcloud", "container", "clusters", "get-credentials", "cluster-0", "--project", "test-project", "--zone", "us-east1"}).Return(nil)
	err := fetchCredentials(c, "test-project", testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)

	// Verify token file
	buf, err := ioutil.ReadFile("/tmp/gcloud.json")
	assert.Equal(t, "{\"key\", \"val\"}", string(buf))

	// Run() error
	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"gcloud", "auth", "activate-service-account", "--key-file", "/tmp/gcloud.json"}).Return(fmt.Errorf("e"))
	err = fetchCredentials(c, "test-project", testRunner)
	testRunner.AssertExpectations(t)
	assert.Error(t, err)
}

func TestTemplateData(t *testing.T) {
	// Set cli.Context with data
	set := flag.NewFlagSet("test-set", 0)
	set.String("drone-branch", "master", "")
	set.String("drone-build-number", "2", "")
	set.String("drone-commit", "e0f21b90a", "")
	set.String("drone-tag", "v0.0.0", "")
	set.String("cluster", "cluster-0", "")
	set.String("zone", "us-east1", "")
	c := cli.NewContext(nil, set, nil)

	// No error
	// Create data maps
	vars := map[string]interface{}{"key0": "val0"}
	secrets := map[string]string{"SECRET_TEST": "test_val"}

	// Call
	tmplData, secretsData, secretsDataRedacted, err := templateData(c, "test-project", vars, secrets)

	// Verify
	assert.Equal(t, map[string]interface{}{
		"BRANCH":       "master",
		"BUILD_NUMBER": "2",
		"COMMIT":       "e0f21b90a",
		"TAG":          "v0.0.0",
		"project":      "test-project",
		"zone":         "us-east1",
		"cluster":      "cluster-0",
		"namespace":    "",
		"key0":         "val0",
	}, tmplData)

	assert.Equal(t, map[string]interface{}{"key0": "val0", "SECRET_TEST": "test_val"}, secretsData)
	assert.Equal(t, map[string]string{"SECRET_TEST": "VALUE REDACTED"}, secretsDataRedacted)
	assert.NoError(t, err)

	// Variable overrides existing ones
	vars = map[string]interface{}{"zone": "us-east4"}
	tmplData, secretsData, secretsDataRedacted, err = templateData(c, "us-east1", vars, secrets)
	assert.Error(t, err)

	// Secret overrides variable
	vars = map[string]interface{}{"SECRET_TEST": "val0"}
	secrets = map[string]string{"SECRET_TEST": "test_val"}
	tmplData, secretsData, secretsDataRedacted, err = templateData(c, "us-east1", vars, secrets)
	assert.Error(t, err)
}

func TestRenderTemplates(t *testing.T) {
	// Mkdir for testing template files
	os.MkdirAll("/tmp/drone-gke-tests/", os.ModePerm)
	kubeTemplatePath := "/tmp/drone-gke-tests/.kube.yml"
	secretTemplatePath := "/tmp/drone-gke-tests/.kube.sec.yml"

	// Set cli.Context with data
	set := flag.NewFlagSet("test-set", 0)
	set.String("kube-template", kubeTemplatePath, "")
	set.String("secret-template", secretTemplatePath, "")
	c := cli.NewContext(nil, set, nil)

	tmplData := map[string]interface{}{
		"COMMIT": "e0f21b90a",
		"key0":   "val0",
	}
	secretsData := map[string]interface{}{"SECRET_TEST": "test_sec_val"}

	// No template file, should error
	os.Remove(kubeTemplatePath)
	os.Remove(secretTemplatePath)
	_, err := renderTemplates(c, tmplData, secretsData)
	assert.Error(t, err)

	// Normal
	// Create test template files
	tmplBuf := []byte("{{.COMMIT}}-{{.key0}}")
	err = ioutil.WriteFile(kubeTemplatePath, tmplBuf, 0600)
	assert.NoError(t, err)
	tmplBuf = []byte("{{.SECRET_TEST}}")
	err = ioutil.WriteFile(secretTemplatePath, tmplBuf, 0600)
	assert.NoError(t, err)

	// Render
	manifestPaths, err := renderTemplates(c, tmplData, secretsData)
	assert.NoError(t, err)

	// Verify token files
	buf, err := ioutil.ReadFile(manifestPaths[kubeTemplatePath])
	assert.Equal(t, "e0f21b90a-val0", string(buf))

	buf, err = ioutil.ReadFile(manifestPaths[secretTemplatePath])
	assert.Equal(t, "test_sec_val", string(buf))

	// Secret variables shouldn't be available in kube template
	tmplBuf = []byte("{{.SECRET_TEST}}")
	err = ioutil.WriteFile(kubeTemplatePath, tmplBuf, 0600)
	_, err = renderTemplates(c, tmplData, secretsData)
	assert.Error(t, err)
}

func TestPrintKubectlVersion(t *testing.T) {
	testRunner := new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "version"}).Return(nil)
	err := printKubectlVersion(testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestSetNamespace(t *testing.T) {
	// No error
	set := flag.NewFlagSet("test-set", 0)
	set.String("zone", "us-east1", "")
	set.String("cluster", "cluster-0", "")
	set.String("namespace", "test-ns", "")
	set.Bool("dry-run", false, "")
	c := cli.NewContext(nil, set, nil)

	testRunner := new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "config", "set-context", "gke_test-project_us-east1_cluster-0", "--namespace", "test-ns"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--filename", "/tmp/namespace.json"}).Return(nil)
	err := setNamespace(c, "test-project", testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)

	// Verify written file
	buf, err := ioutil.ReadFile("/tmp/namespace.json")
	assert.Equal(t, "\n---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: test-ns\n", string(buf))

	// Dry-run
	set = flag.NewFlagSet("test-set", 0)
	set.String("zone", "us-east1", "")
	set.String("cluster", "cluster-0", "")
	set.String("namespace", "test-ns", "")
	set.Bool("dry-run", true, "")
	c = cli.NewContext(nil, set, nil)

	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "config", "set-context", "gke_test-project_us-east1_cluster-0", "--namespace", "test-ns"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--dry-run", "--filename", "/tmp/namespace.json"}).Return(nil)
	err = setNamespace(c, "test-project", testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestApplyManifests(t *testing.T) {
	// Normal
	set := flag.NewFlagSet("test-set", 0)
	set.String("kube-template", ".kube.yml", "")
	set.String("secret-template", ".kube.sec.yml", "")
	set.Bool("dry-run", false, "")
	c := cli.NewContext(nil, set, nil)

	manifestPaths := map[string]string{
		".kube.yml":     "/path/to/kube-tamplate",
		".kube.sec.yml": "/path/to/secret-tamplate",
	}

	testRunner := new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--dry-run", "--filename", "/path/to/kube-tamplate"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--dry-run", "--filename", "/path/to/secret-tamplate"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--filename", "/path/to/kube-tamplate"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--filename", "/path/to/secret-tamplate"}).Return(nil)
	err := applyManifests(c, manifestPaths, testRunner, testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)

	// No secrets manifest
	manifestPaths = map[string]string{
		".kube.yml": "/path/to/kube-tamplate",
	}

	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--dry-run", "--filename", "/path/to/kube-tamplate"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--filename", "/path/to/kube-tamplate"}).Return(nil)
	err = applyManifests(c, manifestPaths, testRunner, testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)

	// Dry-run
	set = flag.NewFlagSet("test-set", 0)
	set.String("kube-template", ".kube.yml", "")
	set.String("secret-template", ".kube.sec.yml", "")
	set.Bool("dry-run", true, "")
	c = cli.NewContext(nil, set, nil)

	manifestPaths = map[string]string{
		".kube.yml":     "/path/to/kube-tamplate",
		".kube.sec.yml": "/path/to/secret-tamplate",
	}

	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--dry-run", "--filename", "/path/to/kube-tamplate"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--dry-run", "--filename", "/path/to/secret-tamplate"}).Return(nil)
	err = applyManifests(c, manifestPaths, testRunner, testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)
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
