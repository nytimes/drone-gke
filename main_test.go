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
	"github.com/urfave/cli/v2"
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
	set := flag.NewFlagSet("empty-set", 0)
	c := cli.NewContext(nil, set, nil)
	err := checkParams(c)
	assert.Error(t, err)

	// Required args set
	set = flag.NewFlagSet("missing-zone-region", 0)
	c = cli.NewContext(nil, set, nil)
	set.String("token", "{}", "")
	set.String("cluster", "cluster-0", "")
	err = checkParams(c)
	assert.Error(t, err)

	// Mutually-exclusive args set
	set = flag.NewFlagSet("both-zone-region", 0)
	c = cli.NewContext(nil, set, nil)
	set.String("token", "{}", "")
	set.String("zone", "us-east1-b", "")
	set.String("region", "us-east1", "")
	set.String("cluster", "cluster-0", "")
	err = checkParams(c)
	assert.Error(t, err)

	// Zonal args set
	set = flag.NewFlagSet("zonal-set", 0)
	c = cli.NewContext(nil, set, nil)
	set.String("token", "{}", "")
	set.String("zone", "us-east1-b", "")
	set.String("cluster", "cluster-0", "")
	err = checkParams(c)
	assert.NoError(t, err)

	// Regional args set
	set = flag.NewFlagSet("regional-set", 0)
	c = cli.NewContext(nil, set, nil)
	set.String("token", "{}", "")
	set.String("region", "us-west1", "")
	set.String("cluster", "cluster-0", "")
	err = checkParams(c)
	assert.NoError(t, err)
}

func TestValidateKubectlVersion(t *testing.T) {
	// kubectl-version is NOT set (using default kubectl version)
	set := flag.NewFlagSet("default-kubectl-version", 0)
	c := cli.NewContext(nil, set, nil)
	availableVersions := []string{}
	err := validateKubectlVersion(c, availableVersions)
	assert.NoError(t, err, "expected validateKubectlVersion to return nil when no kubectl-version param was set")

	// kubectl-version is set and extra kubectl versions are NOT available
	set = flag.NewFlagSet("kubectl-version-set-no-extra-versions-available", 0)
	c = cli.NewContext(nil, set, nil)
	set.String("kubectl-version", "1.14", "")
	availableVersions = []string{}
	err = validateKubectlVersion(c, availableVersions)
	assert.Error(t, err, "expected validateKubectlVersion to return an error when no extra kubectl versions are available")

	// kubectl-version is set, extra kubectl versions are available, kubectl-version is included
	set = flag.NewFlagSet("valid-kubectl-version", 0)
	c = cli.NewContext(nil, set, nil)
	set.String("kubectl-version", "1.13", "")
	availableVersions = []string{"1.11", "1.12", "1.13", "1.14"}
	err = validateKubectlVersion(c, availableVersions)
	assert.NoError(t, err, "expected validateKubectlVersion to return an error when kubectl-version is set, extra kubectl versions are available, kubectl-version is included")

	// kubectl-version is set, extra kubectl versions are available, kubectl-version is NOT included
	set = flag.NewFlagSet("invalid-kubectl-version", 0)
	c = cli.NewContext(nil, set, nil)
	set.String("kubectl-version", "9.99", "")
	availableVersions = []string{"1.11", "1.12", "1.13", "1.14"}
	err = validateKubectlVersion(c, availableVersions)
	assert.Error(t, err, "expected validateKubectlVersion to return nil when kubectl-version is set, extra kubectl versions are available, kubectl-version is NOT included")
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
	zonal := flag.NewFlagSet("zonal-set", 0)
	zonal.String("token", "{\"key\", \"val\"}", "")
	zonal.String("cluster", "cluster-0", "")
	zonal.String("zone", "us-east1-b", "")
	zonalContext := cli.NewContext(nil, zonal, nil)

	regional := flag.NewFlagSet("regional-set", 0)
	regional.String("token", "{\"key\", \"val\"}", "")
	regional.String("cluster", "cluster-0", "")
	regional.String("region", "us-west1", "")
	regionalContext := cli.NewContext(nil, regional, nil)

	// No error
	testRunner := new(MockedRunner)
	testRunner.On("Run", []string{"gcloud", "auth", "activate-service-account", "--key-file", "/tmp/gcloud.json"}).Return(nil)
	testRunner.On("Run", []string{"gcloud", "container", "clusters", "get-credentials", "cluster-0", "--project", "test-project", "--zone", "us-east1-b"}).Return(nil)
	testRunner.On("Run", []string{"gcloud", "container", "clusters", "get-credentials", "cluster-0", "--project", "test-project", "--region", "us-west1"}).Return(nil)
	zonalErr := fetchCredentials(zonalContext, "test-project", testRunner)
	regionalErr := fetchCredentials(regionalContext, "test-project", testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, zonalErr)
	assert.NoError(t, regionalErr)

	// Verify token file
	buf, err := ioutil.ReadFile("/tmp/gcloud.json")
	assert.Equal(t, "{\"key\", \"val\"}", string(buf))

	// Run() error
	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"gcloud", "auth", "activate-service-account", "--key-file", "/tmp/gcloud.json"}).Return(fmt.Errorf("e"))
	err = fetchCredentials(zonalContext, "test-project", testRunner)
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
	set.String("zone", "us-east1-b", "")
	c := cli.NewContext(nil, set, nil)

	// No error
	// Create data maps
	vars := map[string]interface{}{"key0": "val0", "key1": "hello $USER"}
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
		"zone":         "us-east1-b",
		"cluster":      "cluster-0",
		"namespace":    "",
		"key0":         "val0",
		"key1":         "hello $USER",
	}, tmplData)

	assert.Equal(t, map[string]interface{}{
		"BRANCH":       "master",
		"BUILD_NUMBER": "2",
		"COMMIT":       "e0f21b90a",
		"TAG":          "v0.0.0",
		"project":      "test-project",
		"zone":         "us-east1-b",
		"cluster":      "cluster-0",
		"namespace":    "",
		"key0":        "val0",
		"key1":        "hello $USER",
		"SECRET_TEST": "test_val",
	}, secretsData)
	assert.Equal(t, map[string]string{"SECRET_TEST": "VALUE REDACTED"}, secretsDataRedacted)
	assert.NoError(t, err)

	// Variable overrides existing ones
	vars = map[string]interface{}{"zone": "us-east4-b"}
	tmplData, secretsData, secretsDataRedacted, err = templateData(c, "us-east1-b", vars, secrets)
	assert.Error(t, err)

	// Secret overrides variable
	vars = map[string]interface{}{"SECRET_TEST": "val0"}
	secrets = map[string]string{"SECRET_TEST": "test_val"}
	tmplData, secretsData, secretsDataRedacted, err = templateData(c, "us-east1-b", vars, secrets)
	assert.Error(t, err)
}

func TestTemplateDataExpandingVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("USER", "drone-user")

	// Set cli.Context with data
	set := flag.NewFlagSet("test-set", 0)
	set.String("drone-branch", "master", "")
	set.String("drone-build-number", "2", "")
	set.String("drone-commit", "e0f21b90a", "")
	set.String("drone-tag", "v0.0.0", "")
	set.String("cluster", "cluster-0", "")
	set.String("zone", "us-east1-b", "")
	set.Bool("expand-env-vars", true, "")
	c := cli.NewContext(nil, set, nil)

	// No error
	// Create data maps
	vars := map[string]interface{}{"key0": "val0", "key1": "hello $USER"}
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
		"zone":         "us-east1-b",
		"cluster":      "cluster-0",
		"namespace":    "",
		"key0":         "val0",
		"key1":         "hello drone-user",
	}, tmplData)

	assert.Equal(t, map[string]interface{}{
		"BRANCH":       "master",
		"BUILD_NUMBER": "2",
		"COMMIT":       "e0f21b90a",
		"TAG":          "v0.0.0",
		"project":      "test-project",
		"zone":         "us-east1-b",
		"cluster":      "cluster-0",
		"namespace":    "",
		"key0":        "val0",
		"key1":        "hello drone-user",
		"SECRET_TEST": "test_val",
	}, secretsData)
	assert.Equal(t, map[string]string{"SECRET_TEST": "VALUE REDACTED"}, secretsDataRedacted)
	assert.NoError(t, err)

	// Variable overrides existing ones
	vars = map[string]interface{}{"zone": "us-east4-b"}
	tmplData, secretsData, secretsDataRedacted, err = templateData(c, "us-east1-b", vars, secrets)
	assert.Error(t, err)

	// Secret overrides variable
	vars = map[string]interface{}{"SECRET_TEST": "val0"}
	secrets = map[string]string{"SECRET_TEST": "test_val"}
	tmplData, secretsData, secretsDataRedacted, err = templateData(c, "us-east1-b", vars, secrets)
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
	secretsData := map[string]interface{}{
		"COMMIT": "e0f21b90a",
		"key0":   "val0",
		"SECRET_TEST": "test_sec_val",
	}

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
	tmplBuf = []byte("{{.COMMIT}}-{{.SECRET_TEST}}")
	err = ioutil.WriteFile(secretTemplatePath, tmplBuf, 0600)
	assert.NoError(t, err)

	// Render
	manifestPaths, err := renderTemplates(c, tmplData, secretsData)
	assert.NoError(t, err)

	// Verify token files
	buf, err := ioutil.ReadFile(manifestPaths[kubeTemplatePath])
	assert.Equal(t, "e0f21b90a-val0", string(buf))

	buf, err = ioutil.ReadFile(manifestPaths[secretTemplatePath])
	assert.Equal(t, "e0f21b90a-test_sec_val", string(buf))

	// Secret variables shouldn't be available in kube template
	tmplBuf = []byte("{{.SECRET_TEST}}")
	err = ioutil.WriteFile(kubeTemplatePath, tmplBuf, 0600)
	_, err = renderTemplates(c, tmplData, secretsData)
	assert.Error(t, err)
}

func TestParseSkips(t *testing.T) {
	kubeTemplatePath := "/tmp/drone-gke-tests/.kube.yml"
	secretTemplatePath := "/tmp/drone-gke-tests/.kube.sec.yml"

	// Test no skip
	set := flag.NewFlagSet("test-set", 0)
	set.String("kube-template", kubeTemplatePath, "")
	set.String("secret-template", secretTemplatePath, "")
	c := cli.NewContext(nil, set, nil)
	err := parseSkips(c)
	assert.NoError(t, err)
	assert.Equal(t, kubeTemplatePath, c.String("kube-template"))
	assert.Equal(t, secretTemplatePath, c.String("secret-template"))

	// Test skipping both
	set.Bool("skip-template", true, "")
	set.Bool("skip-secret-template", true, "")
	c = cli.NewContext(nil, set, nil)
	err = parseSkips(c)
	assert.Error(t, err)

	// Test skip template
	kubeSet := flag.NewFlagSet("kube-set", 0)
	kubeSet.String("kube-template", kubeTemplatePath, "")
	kubeSet.String("secret-template", secretTemplatePath, "")
	kubeSet.Bool("skip-template", true, "")
	c = cli.NewContext(nil, kubeSet, nil)
	err = parseSkips(c)
	assert.NoError(t, err)
	assert.Empty(t, c.String("kube-template"))
	assert.Equal(t, secretTemplatePath, c.String("secret-template"))

	// Test skip template
	secretSet := flag.NewFlagSet("secret-set", 0)
	secretSet.String("kube-template", kubeTemplatePath, "")
	secretSet.String("secret-template", secretTemplatePath, "")
	secretSet.Bool("skip-secret-template", true, "")
	c = cli.NewContext(nil, secretSet, nil)
	err = parseSkips(c)
	assert.NoError(t, err)
	assert.Equal(t, kubeTemplatePath, c.String("kube-template"))
	assert.Empty(t, c.String("secret-template"))
}

func TestPrintKubectlVersion(t *testing.T) {
	testRunner := new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "version"}).Return(nil)
	err := printKubectlVersion(testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestSetNamespace(t *testing.T) {
	// Zonal args set
	set := flag.NewFlagSet("zonal-set", 0)
	set.String("zone", "us-east1-b", "")
	set.String("cluster", "cluster-0", "")
	set.String("namespace", "test-ns", "")
	set.Bool("dry-run", false, "")
	c := cli.NewContext(nil, set, nil)

	testRunner := new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "config", "set-context", "gke_test-project_us-east1-b_cluster-0", "--namespace", "test-ns"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--filename", "/tmp/namespace.json"}).Return(nil)
	err := setNamespace(c, "test-project", testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)

	// Region args set
	set = flag.NewFlagSet("regional-set", 0)
	set.String("region", "us-west1", "")
	set.String("cluster", "regional-cluster", "")
	set.String("namespace", "test-ns", "")
	set.Bool("dry-run", false, "")
	c = cli.NewContext(nil, set, nil)

	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "config", "set-context", "gke_test-project_us-west1_regional-cluster", "--namespace", "test-ns"}).Return(nil)
	testRunner.On("Run", []string{"kubectl", "apply", "--record", "--filename", "/tmp/namespace.json"}).Return(nil)
	err = setNamespace(c, "test-project", testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)

	// Verify written file
	buf, err := ioutil.ReadFile("/tmp/namespace.json")
	assert.Equal(t, "\n---\napiVersion: v1\nkind: Namespace\nmetadata:\n  name: test-ns\n", string(buf))

	// Dry-run
	set = flag.NewFlagSet("test-set", 0)
	set.String("zone", "us-east1-b", "")
	set.String("cluster", "cluster-0", "")
	set.String("namespace", "Feature/1892-TEST-NS", "")
	set.Bool("dry-run", true, "")
	c = cli.NewContext(nil, set, nil)

	testRunner = new(MockedRunner)
	testRunner.On("Run", []string{"kubectl", "config", "set-context", "gke_test-project_us-east1-b_cluster-0", "--namespace", "feature-1892-test-ns"}).Return(nil)
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

// RunWaitForRollout is a helper function for testing WaitForRollout.  For each flag-value
// in flagValues it will expect a corresponding call of the form:
// "kubectl rollout status <expected-value> ..."
func RunWaitForRollout(t *testing.T, specs []string, expectedValues []string) {
	set := flag.NewFlagSet("test-set", 0)
	set.Int("wait-seconds", 256, "")
	set.String("namespace", "test-ns", "")
	strSlice := cli.StringSlice{}
	for _, spec := range specs {
		strSlice.Set(spec)
	}
	strSliceFlag := cli.StringSliceFlag{Name: "wait-deployments", Value: &strSlice}
	strSliceFlag.Apply(set)
	c := cli.NewContext(nil, set, nil)
	testRunner := new(MockedRunner)
	for _, s := range expectedValues {
		testRunner.On("Run", []string{"timeout", "256", "kubectl", "rollout", "status", s, "--namespace", "test-ns"}).Return(nil)
	}
	err := waitForRollout(c, testRunner)
	testRunner.AssertExpectations(t)
	assert.NoError(t, err)
}

func TestWaitForRollout(t *testing.T) {
	RunWaitForRollout(t, []string{"deployment/d1"}, []string{"deployment/d1"})
	RunWaitForRollout(t,
		[]string{"deployment/d1", "statefulset/s1"},
		[]string{"deployment/d1", "statefulset/s1"})
	RunWaitForRollout(t, []string{"d1"}, []string{"deployment/d1"})
	RunWaitForRollout(t,
		[]string{"d1", "d2"},
		[]string{"deployment/d1", "deployment/d2"})
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

func TestTokenParamPrecedence(t *testing.T) {
	for _, tst := range []struct {
		name           string
		envToken       string
		envPluginToken string

		expectedToken string
		expectedOk    bool
	}{
		{
			name:           "just-plugin-token",
			envToken:       "",
			envPluginToken: "token123",

			expectedOk:    true,
			expectedToken: "token123",
		},
		{
			name:           "just-token",
			envToken:       "token456",
			envPluginToken: "",

			expectedOk:    true,
			expectedToken: "token456",
		},
		{
			name:           "both-and-plugin-token-wins",
			envToken:       "token456",
			envPluginToken: "token123",
			expectedOk:     true,
			expectedToken:  "token123",
		},
		{
			name:           "missing-token",
			envToken:       "",
			envPluginToken: "",

			expectedOk:    false,
			expectedToken: "",
		},
	} {
		t.Run(tst.name, func(t *testing.T) {
			os.Clearenv()

			os.Setenv("PLUGIN_REGION", "region-123")
			os.Setenv("PLUGIN_CLUSTER", "cluster-123")

			if tst.envToken != "" {
				os.Setenv("TOKEN", tst.envToken)
			}

			if tst.envPluginToken != "" {
				os.Setenv("PLUGIN_TOKEN", tst.envPluginToken)
			}

			appErr := (&cli.App{
				Flags: getAppFlags(),
				Action: func(ctx *cli.Context) error {
					if foundToken := ctx.String("token"); foundToken != tst.expectedToken {
						return fmt.Errorf("found token: %s, expected: %s", foundToken, tst.expectedToken)
					}
					return checkParams(ctx)
				},
			}).Run([]string{"run"})

			if tst.expectedOk && appErr != nil {
				t.Fatalf("expected expectedOk, got appErr: %s", appErr)
			} else if !tst.expectedOk && appErr == nil {
				t.Fatalf("expected failure, got appErr: %s", appErr)
			}
		})
	}
}
