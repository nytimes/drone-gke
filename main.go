package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/urfave/cli"
)

var (
	// Build revision is set at compile time.
	rev string
)

const (
	gcloudCmd  = "gcloud"
	kubectlCmd = "kubectl"

	keyPath = "/tmp/gcloud.json"
	nsPath  = "/tmp/namespace.json"
)

var nsTemplate = `
---
apiVersion: v1
kind: Namespace
metadata:
  name: %s
`

func main() {
	err := wrapMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func wrapMain() error {
	if rev == "" {
		rev = "[unknown]"
	}

	fmt.Printf("Drone GKE Plugin built from %s\n", rev)

	app := cli.NewApp()
	app.Name = "gke plugin"
	app.Usage = "gke plugin"
	app.Action = run
	app.Version = fmt.Sprintf("1.0.0-%s", rev)
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "dry-run",
			Usage:  "do not apply the Kubernetes manifests to the API server",
			EnvVar: "PLUGIN_DRY_RUN",
		},
		cli.BoolFlag{
			Name:   "verbose",
			Usage:  "dump available vars and the generated Kubernetes manifest, keeping secrets hidden",
			EnvVar: "PLUGIN_VERBOSE",
		},
		cli.StringFlag{
			Name:   "token",
			Usage:  "service account's JSON credentials",
			EnvVar: "TOKEN",
		},
		cli.StringFlag{
			Name:   "project",
			Usage:  "GCP project name",
			EnvVar: "PLUGIN_PROJECT",
		},
		cli.StringFlag{
			Name:   "zone",
			Usage:  "zone of the container cluster",
			EnvVar: "PLUGIN_ZONE",
		},
		cli.StringFlag{
			Name:   "cluster",
			Usage:  "name of the container cluster",
			EnvVar: "PLUGIN_CLUSTER",
		},
		cli.StringFlag{
			Name:   "namespace",
			Usage:  "Kubernetes namespace to operate in",
			EnvVar: "PLUGIN_NAMESPACE",
		},
		cli.StringFlag{
			Name:   "input-dir",
			Usage:  "optional - input directory with templates for Kubernetes resources",
			EnvVar: "PLUGIN_INPUT_DIR",
			Value:  ".kube/",
		},
		cli.StringFlag{
			Name:   "output-dir",
			Usage:  "optional - output directory for rendered manifests for Kubernetes resources",
			EnvVar: "PLUGIN_OUTPUT_DIR",
			Value:  ".kube-out/",
		},
		cli.StringFlag{
			Name:   "vars",
			Usage:  "variables to use while templating manifests",
			EnvVar: "PLUGIN_VARS",
		},
		cli.StringFlag{
			Name:   "drone-build-number",
			Usage:  "Drone build number",
			EnvVar: "DRONE_BUILD_NUMBER",
		},
		cli.StringFlag{
			Name:   "drone-commit",
			Usage:  "Git commit hash",
			EnvVar: "DRONE_COMMIT",
		},
		cli.StringFlag{
			Name:   "drone-branch",
			Usage:  "Git branch",
			EnvVar: "DRONE_BRANCH",
		},
		cli.StringFlag{
			Name:   "drone-tag",
			Usage:  "Git tag",
			EnvVar: "DRONE_TAG",
		},
	}

	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil
}

func run(c *cli.Context) error {
	// Check required params.

	// Trim whitespace, to forgive the vagaries of YAML parsing.
	token := strings.TrimSpace(c.String("token"))
	if token == "" {
		return fmt.Errorf("Missing required param: token")
	}

	// Use project if explicitly stated, otherwise infer from the service account token.
	project := c.String("project")
	if project == "" {
		project = getProjectFromToken(token)
		if project == "" {
			return fmt.Errorf("Missing required param: project")
		}
	}

	if c.String("zone") == "" {
		return fmt.Errorf("Missing required param: zone")
	}

	// Parse variables.
	vars := make(map[string]interface{})
	varsJSON := c.String("vars")
	if varsJSON != "" {
		if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
			return fmt.Errorf("Error parsing vars: %s\n", err)
		}
	}

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
			return fmt.Errorf("Error: secret key and value mismatch, expected 2, got %d", len(pair))
		}

		k := pair[0]
		v := pair[1]

		if _, ok := secrets[k]; ok {
			return fmt.Errorf("Error: secret var %q shadows existing secret\n", k)
		}

		if v == "" {
			return fmt.Errorf("Error: secret var %q is an empty string\n", k)
		}

		if strings.HasPrefix(k, "SECRET_BASE64_") {
			secrets[k] = v
		} else {
			// Base64 encode secret strings for Kubernetes.
			secrets[k] = base64.StdEncoding.EncodeToString([]byte(v))
		}

		os.Unsetenv(k)
	}

	// Write credentials to tmp file to be picked up by the 'gcloud' command.
	// This is inside the ephemeral plugin container, not on the host.
	err := ioutil.WriteFile(keyPath, []byte(token), 0600)
	if err != nil {
		return fmt.Errorf("Error writing token file: %s\n", err)
	}

	// Warn if the keyfile can't be deleted, but don't abort.
	// We're almost certainly running inside an ephemeral container, so the file will be discarded when we're finished anyway.
	defer func() {
		err := os.Remove(keyPath)
		if err != nil {
			log("Warning: error removing token file: %s\n", err)
		}
	}()

	// Set up the execution environment.
	e := os.Environ()
	e = append(e, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", keyPath))

	runner := NewEnviron("", e, os.Stdout, os.Stderr)

	err = runner.Run(gcloudCmd, "auth", "activate-service-account", "--key-file", keyPath)
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	err = runner.Run(gcloudCmd, "container", "clusters", "get-credentials", c.String("cluster"), "--project", project, "--zone", c.String("zone"))
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Set up the variables available to templates.
	data := map[string]interface{}{
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

	secretsAndData := map[string]interface{}{}
	secretsAndDataKeys := map[string]string{}

	// Add variables to data used for rendering all templates.
	for k, v := range vars {
		// Don't allow vars to be overridden.
		// We do this to ensure that the built-in template vars (above) can be relied upon.
		if _, ok := data[k]; ok {
			return fmt.Errorf("Error: var %q shadows existing var\n", k)
		}

		data[k] = v
		secretsAndData[k] = v
	}

	// Add secrets to data used for rendering the secret (.sec) templates.
	for k, v := range secrets {
		// Don't allow vars to be overridden.
		// We do this to ensure that the built-in template vars (above) can be relied upon.
		if _, ok := secretsAndData[k]; ok {
			return fmt.Errorf("Error: secret var %q shadows existing var\n", k)
		}

		secretsAndData[k] = v
		secretsAndDataKeys[k] = "VALUE REDACTED"
	}

	if c.Bool("verbose") {
		dumpData(os.Stdout, "VARIABLES AVAILABLE FOR ALL TEMPLATES", data)
		dumpData(os.Stdout, "ADDITIONAL SECRET VARIABLES AVAILABLE FOR .sec.yml TEMPLATES", secretsAndDataKeys)
	}

	inputDir := c.String("input-dir")
	outputDir := c.String("output-dir")

	// Ensure output directory does not exist.
	_, err = os.Stat(outputDir)
	if err == nil {
		return fmt.Errorf("Error: output directory %s already exists, will not pollute existing directory\n", outputDir)
	}

	// Create the output directory.
	if os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("Error: unable to create output directory %s for rendered manifests\n", outputDir)
		}
	} else {
		return fmt.Errorf("Error creating output directory: %s\n", err)
	}

	// Loop over all files in input directory.
	files, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("Error reading templates from input directory: %s\n", err)
	}

	for _, f := range files {
		if f.IsDir() {
			// Skip sub-directories.
			continue
		}

		filename := f.Name()
		templateData := map[string]interface{}{}

		switch {
		case strings.HasSuffix(filename, ".sec.yml"):
			// Generate the manifest with `secretsAndData`.
			templateData = secretsAndData
		case strings.HasSuffix(filename, ".yml"):
			// Generate the manifest with `data`.
			templateData = data
		default:
			log("Warning: skipped rendering %s because it is not a .sec.yml or .yml file\n", filename)
			continue
		}

		inName := filepath.Join(inputDir, filename)
		outName := filepath.Join(outputDir, filename)

		// Read the template.
		blob, err := ioutil.ReadFile(inName)
		if err != nil {
			return fmt.Errorf("Error reading template: %s\n", err)
		}

		// Parse the template.
		tmpl, err := template.New(outName).Option("missingkey=error").Parse(string(blob))
		if err != nil {
			return fmt.Errorf("Error parsing template: %s\n", err)
		}

		// Create the output file.
		f, err := os.Create(outName)
		if err != nil {
			return fmt.Errorf("Error creating output file: %s\n", err)
		}

		// Generate the output file.
		err = tmpl.Execute(f, templateData)
		if err != nil {
			return fmt.Errorf("Error rendering manifest from template: %s\n", err)
		}

		f.Close()
	}

	if c.Bool("verbose") {
		for _, f := range files {
			if f.IsDir() {
				// Skip sub-directories.
				continue
			}

			filename := f.Name()
			outName := filepath.Join(outputDir, filename)

			switch {
			case strings.HasSuffix(filename, ".sec.yml"):
				log("Skipped dumping %s because it contains secrets\n", outName)
			case strings.HasSuffix(filename, ".yml"):
				dumpFile(os.Stdout, fmt.Sprintf("RENDERED MANIFEST (%s)", outName), outName)
			}
		}
	}

	// Print kubectl version.
	err = runner.Run(kubectlCmd, "version")
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	namespace := c.String("namespace")

	if namespace != "" {
		// Set the execution namespace.
		log("Configuring kubectl to the %s namespace\n", namespace)

		context := strings.Join([]string{"gke", project, c.String("zone"), c.String("cluster")}, "_")
		err = runner.Run(kubectlCmd, "config", "set-context", context, "--namespace", namespace)
		if err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}

		// Write the namespace manifest to a tmp file for application.
		resource := fmt.Sprintf(nsTemplate, namespace)

		err := ioutil.WriteFile(nsPath, []byte(resource), 0600)
		if err != nil {
			return fmt.Errorf("Error writing namespace resource file: %s\n", err)
		}

		// Ensure the namespace exists, without errors (unlike `kubectl create namespace`).
		log("Ensuring the %s namespace exists\n", namespace)

		nsArgs := applyArgs(c.Bool("dry-run"), nsPath)
		err = runner.Run(kubectlCmd, nsArgs...)
		if err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}
	}

	// If it is not a dry run, do a dry run first to validate Kubernetes manifests.
	log("Validating Kubernetes manifests with a dry-run\n")

	if !c.Bool("dry-run") {
		args := applyArgs(true, outputDir)
		err = runner.Run(kubectlCmd, args...)
		if err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}

		log("Applying Kubernetes manifests to the cluster\n")
	}

	// Actually apply Kubernetes manifests.

	args := applyArgs(c.Bool("dry-run"), outputDir)
	err = runner.Run(kubectlCmd, args...)
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	return nil
}

type token struct {
	ProjectID string `json:"project_id"`
}

func getProjectFromToken(j string) string {
	t := token{}
	err := json.Unmarshal([]byte(j), &t)
	if err != nil {
		return ""
	}
	return t.ProjectID
}

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

func log(format string, a ...interface{}) {
	fmt.Printf("\n"+format, a...)
}
