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
	rev string
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
			Usage:  "do not apply the Kubernetes templates",
			EnvVar: "PLUGIN_DRY_RUN",
		},
		cli.BoolFlag{
			Name:   "verbose",
			Usage:  "dry run disables docker push",
			EnvVar: "PLUGIN_VERBOSE",
		},
		cli.StringFlag{
			Name:   "token",
			Usage:  "service account's JSON credentials",
			EnvVar: "TOKEN",
		},
		cli.StringFlag{
			Name:   "gcloud-cmd",
			Usage:  "alternative gcloud cmd",
			EnvVar: "PLUGIN_GCLOUD_CMD",
		},
		cli.StringFlag{
			Name:   "kubectl-cmd",
			Usage:  "alternative kubectl cmd",
			EnvVar: "PLUGIN_KUBECTL_CMD",
		},
		cli.StringFlag{
			Name:   "project",
			Usage:  "gcp project",
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
			EnvVar: "PLUGIN_NAMEPSACE",
		},
		cli.StringFlag{
			Name:   "kube-template",
			Usage:  "optional - template for e.g. deployments",
			EnvVar: "PLUGIN_TEMPLATE",
			Value:  ".kube.yml",
		},
		cli.StringFlag{
			Name:   "secret-template",
			Usage:  "optional - template for the secret object",
			EnvVar: "PLUGIN_SECRET_TEMPLATE",
			Value:  ".kube.sec.yml",
		},
		cli.StringFlag{
			Name:   "vars",
			Usage:  "variables to use in template",
			EnvVar: "PLUGIN_VARS",
		},
		cli.StringFlag{
			Name:   "drone-build-number",
			Usage:  "variables to use in secret_template. These should already be base64 encoded; the plugin will not do so.",
			EnvVar: "DRONE_BUILD_NUMBER",
		},
		cli.StringFlag{
			Name:   "drone-commit",
			Usage:  "variables to use in secret_template. These should already be base64 encoded; the plugin will not do so.",
			EnvVar: "DRONE_COMMIT",
		},
		cli.StringFlag{
			Name:   "drone-branch",
			Usage:  "variables to use in secret_template. These should already be base64 encoded; the plugin will not do so.",
			EnvVar: "DRONE_BRANCH",
		},
		cli.StringFlag{
			Name:   "drone-tag",
			Usage:  "variables to use in secret_template. These should already be base64 encoded; the plugin will not do so.",
			EnvVar: "DRONE_TAG",
		},
	}

	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil
}

func run(c *cli.Context) error {
	varsJson := c.String("vars")
	vars := make(map[string]interface{})
	if varsJson != "" {
		if err := json.Unmarshal([]byte(varsJson), &vars); err != nil {
			panic(err)
		}
	}

	secrets := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		k := pair[0]
		v := pair[1]
		if _, ok := secrets[k]; ok {
			return fmt.Errorf("Error: secret and base64 secret name conflict %q\n", k)
		} else if v == "" {
			return fmt.Errorf("Error: secret var %q is an empty string\n", k)
		}
		if strings.HasPrefix(k, "SECRET_BASE64_") {
			secrets[k] = v
			os.Unsetenv(k)
		} else if strings.HasPrefix(k, "SECRET_") {
			secrets[k] = base64.StdEncoding.EncodeToString([]byte(v))
			os.Unsetenv(k)
 		}
 	}

	// Check required params.

	// Trim whitespace, to forgive the vagaries of YAML parsing.
	token := strings.TrimSpace(c.String("token"))
	if token == "" {
		return fmt.Errorf("Missing required param: token")
	}

	project := getProjectFromToken(token)
	if project == "" {
		return fmt.Errorf("Missing required param: project")
	}

	if c.String("zone") == "" {
		return fmt.Errorf("Missing required param: zone")
	}

	sdkPath := "/google-cloud-sdk"
	keyPath := "/tmp/gcloud.json"

	// Defaults.
	gcloudCmd := c.String("gcloud-cmd")
	if gcloudCmd == "" {
		gcloudCmd = fmt.Sprintf("%s/bin/gcloud", sdkPath)
	}

	kubectlCmd := c.String("kubectl-cmd")
	if kubectlCmd == "" {
		kubectlCmd = fmt.Sprintf("%s/bin/kubectl", sdkPath)
	}

	kubeTemplate := c.String("kube-template")
	if kubeTemplate == "" {
		kubeTemplate = ".kube.yml"
	}

	secretTemplate := c.String("secret-template")
	if secretTemplate == "" {
		secretTemplate = ".kube.sec.yml"
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
			fmt.Printf("Warning: error removing token file: %s\n", err)
		}
	}()

	e := os.Environ()
	e = append(e, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", keyPath))
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Error while getting working directory: %s\n", err)
	}
	runner := NewEnviron(wd, e, os.Stdout, os.Stderr)

	err = runner.Run(gcloudCmd, "auth", "activate-service-account", "--key-file", keyPath)
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	err = runner.Run(gcloudCmd, "container", "clusters", "get-credentials", c.String("cluster"), "--project", project, "--zone", c.String("zone"))
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	data := map[string]interface{}{
		"BUILD_NUMBER": c.String("drone-build-number"),
		"COMMIT":       c.String("drone-commit"),
		"BRANCH":       c.String("drone-branch"),
		"TAG":          c.String("drone-tag"),

		// https://godoc.org/github.com/drone/drone-plugin-go/plugin#Workspace
		// TODO do we really need these?
		// "workspace": workspace,
		// "repo":      repo,
		// "build":     build,
		// "system":    system,

		// Misc useful stuff.
		// note that secrets like gcp token are excluded
		"project":   project,
		"zone":      c.String("zone"),
		"cluster":   c.String("cluster"),
		"namespace": c.String("namespace"),
	}

	secretsAndData := map[string]interface{}{}
	for k, v := range vars {
		// Don't allow vars to be overridden.
		// We do this to ensure that the built-in template vars (above) can be relied upon.
		if _, ok := data[k]; ok {
			return fmt.Errorf("Error: var %q shadows existing var\n", k)
		}

		data[k] = v
		secretsAndData[k] = v
	}

	if c.Bool("verbose") {
		dump := data
		delete(dump, "workspace")
		dumpData(os.Stdout, "DATA (Workspace Values Omitted)", dump)
	}

	for k, v := range secrets {
		if _, ok := secretsAndData[k]; ok {
			return fmt.Errorf("Error: secret var %q shadows existing var\n", k)
		}
		if v == "" {
			return fmt.Errorf("Error: secret var %q is an empty string\n", k)
		}

		// Base64 encode secret strings.
		secretsAndData[k] = v
	}

	mapping := map[string]map[string]interface{}{
		kubeTemplate:   data,
		secretTemplate: secretsAndData,
	}

	outPaths := make(map[string]string)
	pathArg := []string{}

	for t, content := range mapping {
		if t == "" {
			continue
		}

		inPath := filepath.Join(wd, t)
		bn := filepath.Base(inPath)

		// Ensure the required template file exists.
		_, err := os.Stat(inPath)
		if os.IsNotExist(err) {
			if t == kubeTemplate {
				return fmt.Errorf("Error finding template: %s\n", err)
			} else {
				fmt.Printf("Warning: skipping optional template %s, it was not found\n", t)
				continue
			}
		}

		// Generate the file.
		blob, err := ioutil.ReadFile(inPath)
		if err != nil {
			return fmt.Errorf("Error reading template: %s\n", err)
		}

		tmpl, err := template.New(bn).Option("missingkey=error").Parse(string(blob))
		if err != nil {
			return fmt.Errorf("Error parsing template: %s\n", err)
		}

		outPaths[t] = fmt.Sprintf("/tmp/%s", bn)
		f, err := os.Create(outPaths[t])
		if err != nil {
			return fmt.Errorf("Error creating deployment file: %s\n", err)
		}

		err = tmpl.Execute(f, content)
		if err != nil {
			return fmt.Errorf("Error executing deployment template: %s\n", err)
		}

		f.Close()

		pathArg = append(pathArg, outPaths[t])
	}

	if c.Bool("verbose") {
		dumpFile(os.Stdout, "DEPLOYMENT (Secret Template Omitted)", outPaths[kubeTemplate])
	}

	if c.Bool("dry-run") {
		fmt.Println("Skipping kubectl apply, because dry_run: true")
		return nil
	}

	// Print kubectl version.
	err = runner.Run(kubectlCmd, "version")
	if err != nil {
		return fmt.Errorf("Error: %s\n", err)
	}

	// Set the execution namespace.
	if len(c.String("namespace")) > 0 {
		fmt.Printf("Configuring kubectl to the %s namespace\n", c.String("namespace"))

		context := strings.Join([]string{"gke", project, c.String("zone"), c.String("cluster")}, "_")

		err = runner.Run(kubectlCmd, "config", "set-context", context, "--namespace", c.String("namespace"))
		if err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}

		resource := fmt.Sprintf(nsTemplate, c.String("namespace"))
		nsPath := "/tmp/namespace.json"

		// Write namespace resource file to tmp file to be picked up by the 'kubectl' command.
		// This is inside the ephemeral plugin container, not on the host.
		err := ioutil.WriteFile(nsPath, []byte(resource), 0600)
		if err != nil {
			return fmt.Errorf("Error writing namespace resource file: %s\n", err)
		}

		// Ensure the namespace exists, without errors (unlike `kubectl create namespace`).
		err = runner.Run(kubectlCmd, "apply", "--filename", nsPath)
		if err != nil {
			return fmt.Errorf("Error: %s\n", err)
		}
	}

	// Apply Kubernetes configuration files.
	err = runner.Run(kubectlCmd, "apply", "--filename", strings.Join(pathArg, ","))
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
