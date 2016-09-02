package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"text/template"

	"github.com/drone/drone-plugin-go/plugin"
)

type GKE struct {
	DryRun     bool                   `json:"dry_run"`
	Verbose    bool                   `json:"verbose"`
	Token      string                 `json:"token"`
	GCloudCmd  string                 `json:"gcloud_cmd"`
	KubectlCmd string                 `json:"kubectl_cmd"`
	Project    string                 `json:"project"`
	Zone       string                 `json:"zone"`
	Cluster    string                 `json:"cluster"`
	Template   string                 `json:"template"`
	Vars       map[string]interface{} `json:"vars"`
}

var (
	rev string
)

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

	// https://godoc.org/github.com/drone/drone-plugin-go/plugin
	workspace := plugin.Workspace{}
	repo := plugin.Repo{}
	build := plugin.Build{}
	system := plugin.System{}
	vargs := GKE{}

	plugin.Param("workspace", &workspace)
	plugin.Param("repo", &repo)
	plugin.Param("build", &build)
	plugin.Param("system", &system)
	plugin.Param("vargs", &vargs)
	plugin.MustParse()

	// Check required params

	if vargs.Token == "" {
		return fmt.Errorf("missing required param: token")
	}

	if vargs.Project == "" {
		return fmt.Errorf("missing required param: project")
	}

	if vargs.Zone == "" {
		return fmt.Errorf("missing required param: zone")
	}

	sdkPath := "/google-cloud-sdk"

	// Defaults

	if vargs.GCloudCmd == "" {
		vargs.GCloudCmd = fmt.Sprintf("%s/bin/gcloud", sdkPath)
	}

	if vargs.KubectlCmd == "" {
		vargs.KubectlCmd = fmt.Sprintf("%s/bin/kubectl", sdkPath)
	}

	// Trim whitespace
	vargs.Token = strings.TrimSpace(vargs.Token)

	// Write credentials to tmp file
	keyPath := "/tmp/gcloud.json"
	err := ioutil.WriteFile(keyPath, []byte(vargs.Token), 0600)
	if err != nil {
		return fmt.Errorf("error writing token file: %s\n", err)
	}
	// Warn if the keyfile can't be deleted, but don't abort. We're almost
	// certainly running inside an ephemeral container, so the file will be
	// discarded when we're finished anyway.
	defer func() {
		err := os.Remove(keyPath)
		if err != nil {
			fmt.Printf("warning: error removing token file: %s\n", err)
		}
	}()

	runner := NewEnviron(workspace.Path, os.Environ(), os.Stdout, os.Stderr)

	// fmt.Println("workspace=%v", workspace)
	// fmt.Println("build=%v", build)
	// fmt.Println("vargs=%v", vargs)

	err = runner.Run(vargs.GCloudCmd, "auth", "activate-service-account", "--key-file", keyPath)
	if err != nil {
		return fmt.Errorf("error: %s\n", err)
	}

	err = runner.Run(vargs.GCloudCmd, "container", "clusters", "get-credentials", vargs.Cluster, "--project", vargs.Project, "--zone", vargs.Zone)
	if err != nil {
		return fmt.Errorf("error: %s\n", err)
	}

	inPath := filepath.Join(workspace.Path, vargs.Template)
	bn := filepath.Base(inPath)

	// Generate the deployment file
	//data := makeDeployment("whatever-deployment", )
	blob, err := ioutil.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("error reading template: %s\n", err)
	}

	tmpl, err := template.New(bn).Option("missingkey=error").Parse(string(blob))
	if err != nil {
		return fmt.Errorf("error parsing template: %s\n", err)
	}

	outPath := fmt.Sprintf("/tmp/%s", bn)
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("error creating deployment file: %s\n", err)
	}

	data := map[string]interface{}{

		// http://readme.drone.io/usage/variables/#string-interpolation:2b8b8ac4006be88c769f5e3fd99b009a
		"BUILD_NUMBER": build.Number,
		"COMMIT":       build.Commit,
		"BRANCH":       build.Branch,
		"TAG":          "", // How?

		// https://godoc.org/github.com/drone/drone-plugin-go/plugin#Workspace
		// Note that we don't include the vargs, since that includes the GCP token.
		"workspace": workspace,
		"repo":      repo,
		"build":     build,
		"system":    system,

		// Misc stuff
		"project": vargs.Project,
		"cluster": vargs.Cluster,
	}

	for k, v := range vargs.Vars {

		// Don't allow vars to be overriden.
		_, ok := data[k]
		if ok {
			return fmt.Errorf("var %q shadows existing var\n", k)
		}

		data[k] = v
	}

	if vargs.Verbose {
		dumpTemplateData(os.Stdout, data)
	}

	err = tmpl.Execute(f, data)
	if err != nil {
		return fmt.Errorf("error executing deployment template: %s\n", err)
	}

	// TODO: Move this to defer
	f.Close()

	if vargs.Verbose {
		dumpDeploymentFile(os.Stdout, outPath)
	}

	if vargs.DryRun {
		fmt.Printf("skipping kubectl apply, because dry_run=true\n")
		return nil
	}

	runner.env = append(runner.env, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", keyPath))
	err = runner.Run(vargs.KubectlCmd, "apply", "--filename", outPath)
	if err != nil {
		return fmt.Errorf("error: %s\n", err)
	}

	return nil
}

func dumpTemplateData(w io.Writer, data interface{}) {
	fmt.Fprintln(w, "---START DATA---")
	defer fmt.Fprintln(w, "---END DATA---")

	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		fmt.Fprintf(w, "error marshalling: %s\n", err)
		return
	}

	w.Write(b)
	fmt.Fprintf(w, "\n")
}

func dumpDeploymentFile(w io.Writer, path string) {
	fmt.Fprintln(w, "---START DEPLOYMENT---")
	defer fmt.Fprintln(w, "---END DEPLOYMENT---")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(w, "error reading file: %s\n", err)
		return
	}

	fmt.Fprintln(w, string(data))
}
