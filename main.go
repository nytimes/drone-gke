package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
)

type token struct {
	ProjectID string `json:"project_id"`
}

var (
	// Version is set at compile time.
	version string
	// Build revision is set at compile time.
	rev string
)

const (
	gcloudCmd      = "gcloud"
	kubectlCmdName = "kubectl"
	timeoutCmd     = "timeout"

	keyPath          = "/tmp/gcloud.json"
	nsPath           = "/tmp/namespace.json"
	templateBasePath = "/tmp"
)

// default to kubectlCmdName, can be overriden via kubectl-version param
var kubectlCmd = kubectlCmdName
var extraKubectlVersions = strings.Split(os.Getenv("EXTRA_KUBECTL_VERSIONS"), " ")
var nsTemplate = `
---
apiVersion: v1
kind: Namespace
metadata:
  name: %s
`
var invalidNameRegex = regexp.MustCompile(`[^a-z0-9\.\-]+`)

func main() {
	err := wrapMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getAppFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "dry-run",
			Usage:   "do not apply the Kubernetes manifests to the API server",
			EnvVars: []string{"PLUGIN_DRY_RUN"},
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Usage:   "dump available vars and the generated Kubernetes manifest, keeping secrets hidden",
			EnvVars: []string{"PLUGIN_VERBOSE"},
		},
		&cli.StringFlag{
			Name:    "token",
			Usage:   "service account's `JSON` credentials",
			EnvVars: []string{"PLUGIN_TOKEN", "TOKEN"},
		},
		&cli.StringFlag{
			Name:    "project",
			Usage:   "GCP project name (default: interpreted from JSON credentials)",
			EnvVars: []string{"PLUGIN_PROJECT"},
		},
		&cli.StringFlag{
			Name:    "zone",
			Usage:   "zone of the container cluster",
			EnvVars: []string{"PLUGIN_ZONE"},
		},
		&cli.StringFlag{
			Name:    "region",
			Usage:   "region of the container cluster",
			EnvVars: []string{"PLUGIN_REGION"},
		},
		&cli.StringFlag{
			Name:    "cluster",
			Usage:   "name of the container cluster",
			EnvVars: []string{"PLUGIN_CLUSTER"},
		},
		&cli.StringFlag{
			Name:    "namespace",
			Usage:   "Kubernetes namespace to operate in",
			EnvVars: []string{"PLUGIN_NAMESPACE"},
		},
		&cli.StringFlag{
			Name:    "kube-template",
			Usage:   "template for Kubernetes resources, e.g. Deployments",
			EnvVars: []string{"PLUGIN_TEMPLATE"},
			Value:   ".kube.yml",
		},
		&cli.BoolFlag{
			Name:    "skip-template",
			Usage:   "do not parse or apply the Kubernetes template",
			EnvVars: []string{"PLUGIN_SKIP_TEMPLATE"},
		},
		&cli.StringFlag{
			Name:    "secret-template",
			Usage:   "template for Kubernetes Secret resources",
			EnvVars: []string{"PLUGIN_SECRET_TEMPLATE"},
			Value:   ".kube.sec.yml",
		},
		&cli.BoolFlag{
			Name:    "skip-secret-template",
			Usage:   "do not parse or apply the Kubernetes Secret template",
			EnvVars: []string{"PLUGIN_SKIP_SECRET_TEMPLATE"},
		},
		&cli.StringFlag{
			Name:    "vars",
			Usage:   "variables to use while templating manifests in `JSON` format",
			EnvVars: []string{"PLUGIN_VARS"},
		},
		&cli.BoolFlag{
			Name:    "expand-env-vars",
			Usage:   "expand environment variables contents on vars",
			EnvVars: []string{"PLUGIN_EXPAND_ENV_VARS"},
		},
		&cli.StringFlag{
			Name:    "drone-build-number",
			Usage:   "Drone build number",
			EnvVars: []string{"DRONE_BUILD_NUMBER"},
		},
		&cli.StringFlag{
			Name:    "drone-commit",
			Usage:   "Git commit hash",
			EnvVars: []string{"DRONE_COMMIT"},
		},
		&cli.StringFlag{
			Name:    "drone-branch",
			Usage:   "Git branch",
			EnvVars: []string{"DRONE_BRANCH"},
		},
		&cli.StringFlag{
			Name:    "drone-tag",
			Usage:   "Git tag",
			EnvVars: []string{"DRONE_TAG"},
		},
		&cli.StringSliceFlag{
			Name:    "wait-deployments",
			Usage:   "list of Deployments to wait for successful rollout using kubectl rollout status in `JSON` format",
			EnvVars: []string{"PLUGIN_WAIT_DEPLOYMENTS"},
		},
		&cli.IntFlag{
			Name:    "wait-seconds",
			Usage:   "if wait-deployments is set, number of seconds to wait before failing the build",
			EnvVars: []string{"PLUGIN_WAIT_SECONDS"},
			Value:   0,
		},
		&cli.StringFlag{
			Name:    "kubectl-version",
			Usage:   "optional - version of kubectl binary to use, e.g. 1.14",
			EnvVars: []string{"PLUGIN_KUBECTL_VERSION"},
		},
	}
}

func wrapMain() error {
	if version == "" {
		version = "x.x.x"
	}

	if rev == "" {
		rev = "[unknown]"
	}

	fmt.Printf("Drone GKE Plugin built from %s\n", rev)

	app := cli.NewApp()
	app.Name = "gke plugin"
	app.Usage = "gke plugin"
	app.Action = run
	app.Version = fmt.Sprintf("%s-%s", version, rev)
	app.Flags = getAppFlags()
	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil
}

func run(c *cli.Context) error {
	// Check required params
	if err := checkParams(c); err != nil {
		return err
	}

	// Use project if explicitly stated, otherwise infer from the service account token.
	project := c.String("project")
	if project == "" {
		log("Parsing Project ID from credentials\n")
		project = getProjectFromToken(c.String("token"))
		if project == "" {
			return fmt.Errorf("Missing required param: project")
		}
	}

	// Parse skipping template processing.
	err := parseSkips(c)
	if err != nil {
		return err
	}

	// Use custom kubectl version if provided.
	kubectlVersion := c.String("kubectl-version")
	if kubectlVersion != "" {
		kubectlCmd = fmt.Sprintf("%s.%s", kubectlCmdName, kubectlVersion)
	}

	// Parse variables and secrets
	vars, err := parseVars(c)
	if err != nil {
		return err
	}

	secrets, err := parseSecrets()
	if err != nil {
		return err
	}

	// Setup execution environment
	environ := os.Environ()
	environ = append(environ, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", keyPath))
	runner := NewBasicRunner("", environ, os.Stdout, os.Stderr)

	// Auth with gcloud and fetch kubectl credentials
	if err := fetchCredentials(c, project, runner); err != nil {
		return err
	}

	// Delete credentials from filesystem when finishing
	// Warn if the keyfile can't be deleted, but don't abort.
	// We're almost certainly running inside an ephemeral container, so the file will be discarded when we're finished anyway.
	defer func() {
		err := os.Remove(keyPath)
		if err != nil {
			log("Warning: error removing token file: %s\n", err)
		}
	}()

	// Build template data maps
	templateData, secretsData, secretsDataRedacted, err := templateData(c, project, vars, secrets)
	if err != nil {
		return err
	}

	// Print variables and secret keys
	if c.Bool("verbose") {
		dumpData(os.Stdout, "VARIABLES AVAILABLE FOR ALL TEMPLATES", templateData)
		dumpData(os.Stdout, "ADDITIONAL SECRET VARIABLES AVAILABLE FOR .sec.yml TEMPLATES", secretsDataRedacted)
	}

	// Render manifest templates
	manifestPaths, err := renderTemplates(c, templateData, secretsData)
	if err != nil {
		return err
	}

	// Print rendered file
	if c.Bool("verbose") {
		dumpFile(os.Stdout, "RENDERED MANIFEST (Secret Manifest Omitted)", manifestPaths[c.String("kube-template")])
	}

	// kubectl version
	if err := printKubectlVersion(runner); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Set namespace and ensure it exists
	if err := setNamespace(c, project, runner); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Apply manifests
	// Separate runner for catching secret output
	var secretStderr bytes.Buffer
	runnerSecret := NewBasicRunner("", environ, os.Stdout, &secretStderr)
	if err := applyManifests(c, manifestPaths, runner, runnerSecret); err != nil {
		// Print last line of error of applying secret manifest to stderr
		// Disable it for now as it might still leak secrets
		// printTrimmedError(&secretStderr, os.Stderr)
		return fmt.Errorf("Error (kubectl output redacted): %s\n", err)
	}

	if c.Bool("dry-run") {
		log("Not waiting for rollout, this was a dry-run\n")
		return nil
	}
	// Wait for rollout to finish
	if err := waitForRollout(c, runner); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

// checkParams checks required params
func checkParams(c *cli.Context) error {
	if c.String("token") == "" {
		return fmt.Errorf("Missing required param: token")
	}

	if c.String("zone") == "" && c.String("region") == "" {
		return fmt.Errorf("Missing required param: at least one of region or zone must be specified")
	}

	if c.String("zone") != "" && c.String("region") != "" {
		return fmt.Errorf("Invalid params: at most one of region or zone may be specified")
	}

	if c.String("cluster") == "" {
		return fmt.Errorf("Missing required param: cluster")
	}

	if err := validateKubectlVersion(c, extraKubectlVersions); err != nil {
		return err
	}

	return nil
}

// validateKubectlVersion tests whether a given version is valid within the current environment
func validateKubectlVersion(c *cli.Context, availableVersions []string) error {
	kubectlVersionParam := c.String("kubectl-version")
	// using the default version
	if kubectlVersionParam == "" {
		return nil
	}

	// using a custom version but no extra versions are available
	if len(availableVersions) == 0 {
		return fmt.Errorf("Invalid param: kubectl-version was set to %s but no extra kubectl versions are available", kubectlVersionParam)
	}

	// using a custom version ...
	// return nil if included in available extra versions; error otherwise
	for _, availableVersion := range availableVersions {
		if kubectlVersionParam == availableVersion {
			return nil
		}
	}
	return fmt.Errorf("Invalid param kubectl-version: %s must be one of %s", kubectlVersionParam, strings.Join(availableVersions, ", "))
}

// getProjectFromToken gets project id from token
func getProjectFromToken(j string) string {
	t := token{}
	err := json.Unmarshal([]byte(j), &t)
	if err != nil {
		return ""
	}
	return t.ProjectID
}

// parseSkips determines which templates will be processed.
// Prior in Drone 0.8 we allowed setting template filenames to an empty string "" to skip processing them.
// As of Drone 1.7, env vars that have an empty string as the value are dropped.
// So we need to use and check the new set of flags to determine if the user wants to skip processing a template file.
func parseSkips(c *cli.Context) error {
	if c.Bool("skip-template") {
		log("Warning: skipping kube-template because it was set to be ignored\n")
		if err := c.Set("kube-template", ""); err != nil {
			return err
		}
	}
	if c.Bool("skip-secret-template") {
		log("Warning: skipping secret-template because it was set to be ignored\n")
		if err := c.Set("secret-template", ""); err != nil {
			return err
		}
	}

	if c.Bool("skip-template") && c.Bool("skip-secret-template") {
		return fmt.Errorf("Error: skipping both templates ends the plugin execution\n")
	}

	return nil
}

// parseVars parses vars (in JSON) and returns a map
func parseVars(c *cli.Context) (map[string]interface{}, error) {
	// Parse variables.
	vars := make(map[string]interface{})
	varsJSON := c.String("vars")
	if varsJSON != "" {
		if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
			return nil, fmt.Errorf("Error parsing vars: %s\n", err)
		}
	}

	return vars, nil
}

// parseSecrets parses secrets from environment variables (beginning with "SECRET_"),
// clears them and returns a map
func parseSecrets() (map[string]string, error) {
	// Parse secrets.
	secrets := make(map[string]string)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "SECRET_") {
			continue
		}

		// Only split up to 2 parts.
		pair := strings.SplitN(e, "=", 2)

		// Check that key and value both exist.
		if len(pair) != 2 {
			return nil, fmt.Errorf("Error: missing secret value")
		}

		k := pair[0]
		v := pair[1]

		if _, ok := secrets[k]; ok {
			return nil, fmt.Errorf("Error: secret var %q shadows existing secret\n", k)
		}

		if v == "" {
			return nil, fmt.Errorf("Error: secret var %q is an empty string\n", k)
		}

		if strings.HasPrefix(k, "SECRET_BASE64_") {
			secrets[k] = v
		} else {
			// Base64 encode secret strings for Kubernetes.
			secrets[k] = base64.StdEncoding.EncodeToString([]byte(v))
		}

		os.Unsetenv(k)
	}

	return secrets, nil
}

// fetchCredentials authenticates with gcloud and fetches credentials for kubectl
func fetchCredentials(c *cli.Context, project string, runner Runner) error {
	// Write credentials to tmp file to be picked up by the 'gcloud' command.
	// This is inside the ephemeral plugin container, not on the host.
	err := ioutil.WriteFile(keyPath, []byte(c.String("token")), 0600)
	if err != nil {
		return fmt.Errorf("Error writing token file: %s\n", err)
	}

	err = runner.Run(gcloudCmd, "auth", "activate-service-account", "--key-file", keyPath)
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	getCredentialsArgs := []string{
		"container",
		"clusters",
		"get-credentials", c.String("cluster"),
		"--project", project,
	}

	// build --zone / --region arguments based on parameters provided to plugin
	// checkParams requires at least one of zone or region to be provided and prevents use of both at the same time
	if c.String("zone") != "" {
		getCredentialsArgs = append(getCredentialsArgs, "--zone", c.String("zone"))
	}

	if c.String("region") != "" {
		getCredentialsArgs = append(getCredentialsArgs, "--region", c.String("region"))
	}

	err = runner.Run(gcloudCmd, getCredentialsArgs...)
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

// templateData builds template and data maps
func templateData(c *cli.Context, project string, vars map[string]interface{}, secrets map[string]string) (map[string]interface{}, map[string]interface{}, map[string]string, error) {
	// Built-in template vars
	templateData := map[string]interface{}{
		"BUILD_NUMBER": c.String("drone-build-number"),
		"COMMIT":       c.String("drone-commit"),
		"BRANCH":       c.String("drone-branch"),
		"TAG":          c.String("drone-tag"),

		// Misc useful stuff.
		// Note that secrets (including the GCP token) are excluded
		"project":   project,
		"zone":      c.String("zone"),
		"cluster":   c.String("cluster"),
		"namespace": c.String("namespace"),
	}

	secretsData := map[string]interface{}{}
	secretsDataRedacted := map[string]string{}

	for k, v := range templateData {
		secretsData[k] = v
	}

	// Add variables to data used for rendering both templates.
	for k, v := range vars {
		// Don't allow vars to be overridden.
		// We do this to ensure that the built-in template vars (above) can be relied upon.
		if _, ok := templateData[k]; ok {
			return nil, nil, nil, fmt.Errorf("Error: var %q shadows existing var\n", k)
		}

		if c.Bool("expand-env-vars") {
			if rawValue, ok := v.(string); ok {
				v = os.ExpandEnv(rawValue)
			}
		}

		templateData[k] = v
		secretsData[k] = v
	}

	// Add secrets to data used for rendering the Secret template.
	for k, v := range secrets {
		// Don't allow vars to be overridden.
		// We do this to ensure that the built-in template vars (above) can be relied upon.
		if _, ok := secretsData[k]; ok {
			return nil, nil, nil, fmt.Errorf("Error: secret var %q shadows existing var\n", k)
		}

		secretsData[k] = v
		secretsDataRedacted[k] = "VALUE REDACTED"
	}

	return templateData, secretsData, secretsDataRedacted, nil
}

// renderTemplates renders templates, writes into files and returns rendered template paths
func renderTemplates(c *cli.Context, templateData map[string]interface{}, secretsData map[string]interface{}) (map[string]string, error) {
	// mapping is a map of the template filename to the data it uses for rendering.
	mapping := map[string]map[string]interface{}{
		c.String("kube-template"):   templateData,
		c.String("secret-template"): secretsData,
	}

	manifestPaths := make(map[string]string)

	// YAML files path for kubectl
	for t, content := range mapping {
		if t == "" {
			continue
		}

		// Ensure the required template file exists.
		_, err := os.Stat(t)
		if os.IsNotExist(err) {
			if t == c.String("kube-template") {
				return nil, fmt.Errorf("Error finding template: %s\n", err)
			}

			log("Warning: skipping optional secret template %s because it was not found\n", t)
			continue
		}

		// Create the output file.
		// If template is a path, extract file name
		filename := filepath.Base(t)
		manifestPaths[t] = path.Join(templateBasePath, filename)
		f, err := os.Create(manifestPaths[t])
		if err != nil {
			return nil, fmt.Errorf("Error creating deployment file: %s\n", err)
		}

		// Read the template.
		blob, err := ioutil.ReadFile(t)
		if err != nil {
			return nil, fmt.Errorf("Error reading template: %s\n", err)
		}

		// Parse the template.
		tmpl, err := template.New(t).Option("missingkey=error").Parse(string(blob))
		if err != nil {
			return nil, fmt.Errorf("Error parsing template: %s\n", err)
		}

		// Generate the manifest.
		err = tmpl.Execute(f, content)
		if err != nil {
			return nil, fmt.Errorf("Error rendering deployment manifest from template: %s\n", err)
		}

		f.Close()
	}

	return manifestPaths, nil
}

// printKubectlVersion runs kubectl version
func printKubectlVersion(runner Runner) error {
	return runner.Run(kubectlCmd, "version")
}

// setNamespace sets namespace of current kubectl context and ensure it exists
func setNamespace(c *cli.Context, project string, runner Runner) error {
	namespace := c.String("namespace")
	if namespace == "" {
		return nil
	}

	//replace invalid char in namespace
	namespace = strings.ToLower(namespace)
	namespace = invalidNameRegex.ReplaceAllString(namespace, "-")

	// Set the execution namespace.
	log("Configuring kubectl to the %s namespace\n", namespace)

	// set cluster location segment based on parameters provided to plugin
	// checkParams requires at least one of zone or region to be provided and prevents use of both at the same time
	clusterLocation := ""
	if c.String("zone") != "" {
		clusterLocation = c.String("zone")
	}

	if c.String("region") != "" {
		clusterLocation = c.String("region")
	}

	context := strings.Join([]string{"gke", project, clusterLocation, c.String("cluster")}, "_")

	if err := runner.Run(kubectlCmd, "config", "set-context", context, "--namespace", namespace); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Write the namespace manifest to a tmp file for application.
	resource := fmt.Sprintf(nsTemplate, namespace)

	if err := ioutil.WriteFile(nsPath, []byte(resource), 0600); err != nil {
		return fmt.Errorf("Error writing namespace resource file: %s\n", err)
	}

	// Ensure the namespace exists, without errors (unlike `kubectl create namespace`).
	log("Ensuring the %s namespace exists\n", namespace)

	nsArgs := applyArgs(c.Bool("dry-run"), nsPath)
	if err := runner.Run(kubectlCmd, nsArgs...); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

// applyManifests applies manifests using kubectl apply
func applyManifests(c *cli.Context, manifestPaths map[string]string, runner Runner, runnerSecret Runner) error {

	manifests := manifestPaths[c.String("kube-template")]
	manifestsSecret := manifestPaths[c.String("secret-template")]

	// If it is not a dry run, do a dry run first to validate Kubernetes manifests.
	log("Validating Kubernetes manifests with a dry-run\n")

	if !c.Bool("dry-run") {
		args := applyArgs(true, manifests)
		if err := runner.Run(kubectlCmd, args...); err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}

		if len(manifestsSecret) > 0 {
			argsSecret := applyArgs(true, manifestsSecret)
			if err := runnerSecret.Run(kubectlCmd, argsSecret...); err != nil {
				return fmt.Errorf("Error: %s\n", err)
			}
		}

		log("Applying Kubernetes manifests to the cluster\n")
	}

	// Actually apply Kubernetes manifests.
	args := applyArgs(c.Bool("dry-run"), manifests)
	if err := runner.Run(kubectlCmd, args...); err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Apply Kubernetes secrets manifests
	if len(manifestsSecret) > 0 {
		argsSecret := applyArgs(c.Bool("dry-run"), manifestsSecret)
		if err := runnerSecret.Run(kubectlCmd, argsSecret...); err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}
	}

	return nil
}

// waitForRollout executes kubectl to wait for rollout to complete before continuing
func waitForRollout(c *cli.Context, runner Runner) error {
	namespace := c.String("namespace")
	waitSeconds := c.Int("wait-seconds")
	specs := c.StringSlice("wait-deployments")
	waitDeployments := []string{}

	for _, spec := range specs {
		// default type to "deployment" if not present
		deployment := spec
		if !strings.Contains(spec, "/") {
			deployment = "deployment/" + deployment
		}
		waitDeployments = append(waitDeployments, deployment)
	}

	waitDeploymentsCount := len(waitDeployments)
	counterProgress := ""

	for counter, deployment := range waitDeployments {
		if waitDeploymentsCount > 1 {
			counterProgress = fmt.Sprintf(" %d/%d", counter+1, waitDeploymentsCount)
		}

		log(fmt.Sprintf("Waiting until rollout completes for %s%s\n", deployment, counterProgress))

		command := []string{"rollout", "status", deployment}

		if namespace != "" {
			command = append(command, "--namespace", namespace)
		}

		path := kubectlCmd

		if waitSeconds != 0 {
			command = append([]string{strconv.Itoa(waitSeconds), path}, command...)
			path = timeoutCmd
		}

		if err := runner.Run(path, command...); err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}
	}

	return nil
}

// applyArgs creates args slice for kubectl apply command
func applyArgs(dryrun bool, file string) []string {
	args := []string{
		"apply",
		"--record",
	}

	if dryrun {
		args = append(args, "--dry-run")
	}

	args = append(args, "--filename")
	args = append(args, file)

	return args
}

// printTrimmedError prints the last line of stderrbuf to dest
func printTrimmedError(stderrbuf io.Reader, dest io.Writer) {
	var lastLine string
	scanner := bufio.NewScanner(stderrbuf)
	for scanner.Scan() {
		lastLine = scanner.Text()
	}
	fmt.Fprintf(dest, "%s\n", lastLine)
}

func log(format string, a ...interface{}) {
	fmt.Printf("\n"+format, a...)
}
