# drone-gke Usage

Use this plugin to deploy Docker images to [Google Container Engine (GKE)][gke].

[gke]: https://cloud.google.com/container-engine/

## API Reference

The following parameters are used to configure this plugin:

### `image`

_**type**_ `string`

_**default**_ `''`

_**description**_ reference to a `drone-gke` Docker image

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke:0.9
    # ...
```

### `namespace`

_**type**_ `string`

_**default**_ `''`

_**description**_ Kubernetes namespace to operate in

_**notes**_ sets the context for kubectl

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      namespace: my-app
      # ...
```

### `project`

_**type**_ `string`

_**default**_ `''`

_**description**_ GCP project ID; owner of the GKE cluster

_**notes**_ default inferred from the service account credentials provided via [`token`](#service-account-credentials)

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      project: my-gcp-project
      # ...
```

### `zone`

_**type**_ `string`

_**default**_ `''`

_**description**_ GCP zone where GKE cluster is located

_**notes**_ required if [`region`](#region) is not provided

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      zone: us-east1-b
      # ...
```

### `region`

_**type**_ `string`

_**default**_ `''`

_**description**_ GCP region where GKE cluster is located

_**notes**_ required if [`zone`](#zone) is not provided

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      region: us-west-1
      # ...
```

### `cluster`

_**type**_ `string`

_**default**_ `''`

_**description**_ name of GKE cluster

_**notes**_ required

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      cluster: prod
      # ...
```

### `namespace`

_**type**_ `string`

_**default**_ `''`

_**description**_ name of Kubernetes Namespace where manifests will be applied

_**notes**_ if not specified, resources will be applied to `default` Namespace; if specified and [`create_namespace`](#create_namespace) is set to `false`, the Namespace resource must already exist within the cluster

_**example**_

```yaml
# .drone.yml
---
pipeline:
  # ...
  deploy:
    image: nytimes/drone-gke
    cluster: prod
    namespace: petstore
    # ...
```

### `template`

_**type**_ `string`

_**default**_ `'.kube.yml'`

_**description**_ path to Kubernetes manifest template

_**notes**_ rendered using the Go [`text/template`](https://golang.org/pkg/text/template/) package.
The `secret_template` manifest (`.kube.sec.yml`) will apply prior to the `template` manifest (`.kube.yml`), as the secrets may need to be available to the main manifest file.
If the file does not exist or you do not want the plugin to apply the template, set `skip_template` to `true`.

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      template: k8s/app.yaml
      # ...
```

### `skip_template`

_**type**_ `bool`

_**default**_ `false`

_**description**_ parse and apply the Kubernetes manifest template

_**notes**_ turn off the use of the template, regardless if the file exists or not

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      template: k8s/app.yaml
      skip_template: true
      # ...
```

### `secret_template`

_**type**_ `string`

_**default**_ `'.kube.sec.yml'`

_**description**_ path to Kubernetes [_Secret_ resource](http://kubernetes.io/docs/user-guide/secrets/) manifest template

_**notes**_ rendered using the Go [`text/template`](https://golang.org/pkg/text/template/) package.
The `secret_template` manifest (`.kube.sec.yml`) will apply prior to the `template` manifest (`.kube.yml`), as the secrets may need to be available to the main manifest file.
If the file does not exist or you do not want the plugin to apply the template, set `skip_secret_template` to `true`.

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      secret_template: my-templates/secrets.yaml
      # ...
```

### `skip_secret_template`

_**type**_ `bool`

_**default**_ `false`

_**description**_ parse and apply the Kubernetes _Secret_ resource manifest template

_**notes**_ turn off the use of the template, regardless if the file exists or not

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      secret_template: my-templates/secrets.yaml
      skip_secret_template: true
      # ...
```

### `wait_deployments`

_**type**_ `[]string`

_**default**_ `[]`

_**description**_ wait for the given deployments using `kubectl rollout status ...`

_**notes**_ deployments can be specified as `"<type>/<name>"` as expected by `kubectl`.  If 
just `"<name>"` is given it will be defaulted to `"deployment/<name>"`.  

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      wait_deployments:
      - deployment/app
      - statefulset/memcache
      - nginx
      # ...
```

### `wait_seconds`

_**type**_ `int`

_**default**_ `0`

_**description**_ number of seconds to wait before failing the build

_**notes**_ ignored if `wait_deployments` is not set

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      wait_seconds: 180
      wait_deployments:
      - deployment/app
      - statefulset/memcache
      - nginx
      # ...
```

### `wait_jobs`

_**type**_ `[]string`

_**default**_ `[]`

_**description**_ wait for the given jobs using `kubectl wait --for=conditon=complete ...`

_**notes**_ deployments can be specified as `"job/<name>"` as expected by `kubectl`.
If just `"<name>"` is given it will be defaulted to `"job/<name>"`.

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      wait_jobs:
      - job/migration
      - otherjob
      # ...
```

### `wait_jobs_seconds`

_**type**_ `int`

_**default**_ `0`

_**description**_ number of seconds to wait for jobs to complete before failing the build

_**notes**_ ignored if `wait_jobs` is not set

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      wait_jobs_seconds: 180
      wait_jobs:
      - migration
      # ...
```

### `vars`

_**type**_ `map[string]interface{}`

_**default**_

```go
{
  // from $DRONE_BUILD_NUMBER
  "BUILD_NUMBER": "",
  // from $DRONE_COMMIT
  "COMMIT": "",
  // from $DRONE_BRANCH
  "BRANCH": "",
  // from $DRONE_TAG
  "TAG": "",
  // from `project`
  "project": "",
  // from `zone`
  "zone": "",
  // from `cluster`
  "cluster": "",
  // from `namespace`
  "namespace": "",
}
```

_**description**_ variables to use in [`template`](#template) and [`secret_template`](#secret_template)

_**notes**_ see ["Available vars"](#available-vars) for details

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      vars:
        app_name: echo
        app_image: gcr.io/google_containers/echoserver:1.4
        env: dev
      # ...
```

### `commands`

_**type**_ `[]string`

_**default**_ `[]`

_**description**_ custom commands that will run inside the cluster

_**notes**_ commands run after the [`secret_template`](#secret_template) and [`template`](#template) manifests are applied and before [`wait_deployments`](#wait_deployments) and [`wait_jobs`](#wait_jobs) run.

commands will not run if [`dry_run`](#dry_run) is set to `true`.

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      commands:
      - echo 'hello'
      - echo 'have a good day'
      # ...
```

### `secrets`

_**type**_ `map[string]string`

_**default**_ `{}`

_**description**_ variables to use in [`secret_template`](#secret_template); credentials for `drone-gke`

_**notes**_ `TOKEN` is required; `SECRET_` prefix is required - see ["Using secrets"](#using-secrets) for details

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      # ...
    environment:
      # custom secrets; only available within `secret_template`
      SECRET_APP_API_KEY:
        from_secret: APP_API_KEY_DEV
      SECRET_BASE64_P12_CERT:
        from_secret: BASE64_P12_CERT_DEV
      # required by `drone-gke`; not available within templates
      TOKEN:
        from_secret: DRONE_GKE_SERVICE_ACCOUNT_KEY_DEV
```

### `expand_env_vars`

_**type**_ `bool`

_**default**_ `false`

_**description**_ expand environment variables for values in [`vars`](#vars) for reference

_**notes**_ only available for `vars` of type `string`; use `$${var_to_expand}` instead of `${var_to_expand}` (two `$$`) to escape drone's variable substitution

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    environment:
      # ;)
      PS1: 'C:${PWD//\//\\\\}>'
    settings:
      expand_env_vars: true
      vars:
        # CLOUD_SDK_VERSION (set by google/cloud-sdk) will be expanded
        message: "deployed using gcloud v$${CLOUD_SDK_VERSION}"
        # PS1 (set using standard drone `environment` field) will be expanded
        prompt: "cmd.exe $${PS1}"
        # HOSTNAME and PATH (set by shell) will not be expanded; the `hostnames` and `env` vars are set to non-string values
        hostnames:
        - example.com
        - blog.example.com
        - "$${HOSTNAME}"
        env:
          path: "$${PATH}"
      # ...
```

### `kubectl_version`

_**type**_ `string`

_**default**_ `''`

_**description**_ version of kubectl executable to use

_**notes**_ see [Using "extra" `kubectl` versions](#using-extra-kubectl-versions) for details

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      kubectl_version: "1.14"
      # ...
```

### `dry_run`

_**type**_ `bool`

_**default**_ `false`

_**description**_ do not apply the Kubernetes templates

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      dry_run: true
      # ...
```

### `server_side`

_**type**_ `bool`

_**default**_ `false`

_**description**_ Perform a Kubernetes [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/)

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      server_side: true
      # ...
```

### `verbose`

_**type**_ `bool`

_**default**_ `false`

_**description**_ dump available [`vars`](#vars) and the generated Kubernetes [`template`](#template)

_**notes**_ excludes secrets

_**example**_

```yaml
# .drone.yml
---
kind: pipeline
# ...
steps:
  - name: deploy-gke
    image: nytimes/drone-gke
    settings:
      verbose: true
      # ...
```

### `create_namespace`

_**type**_ `bool`

_**default**_ `true`

_**description**_ automatically create a Namespace resource when a [`namespace`](#namespace) value is specified

_**notes**_ depends on non-empty `namespace` value; the resource will _always_ be applied to the cluster _prior to_ any resources included in [`template`](#template) / [`secret_template`](#secret_template); may modify any existing Namespace resource configuration; the automatically created Namespace resource is not configurable (see [source](https://github.com/nytimes/drone-gke/blob/2909135b2dce136aa5095d609d91b0963fbb4697/main.go#L51-L54) for more details);

_**example**_

```yaml
# .drone.yml
---
pipeline:
  # ...
  deploy:
    image: nytimes/drone-gke
    namespace: petstore
    create_namespace: false
    # ...
```

## Service Account Credentials

`drone-gke` requires a Google service account and uses its [JSON credential file][service-account] to authenticate.

This must be passed to the plugin under the target `token`. If provided in base64 format, the plugin will decode it internally.

The plugin infers the GCP project from the JSON credentials (`token`) and retrieves the GKE cluster credentials.

[service-account]: https://cloud.google.com/storage/docs/authentication#service_accounts

#### GUI

Simply copy the contents of the JSON credentials file and paste it directly in the input field (for example for a secret named `GOOGLE_CREDENTIALS`).

#### CLI

```sh
drone secret add \
  --event push \
  --event pull_request \
  --event tag \
  --event deployment \
  --repository org/repo \
  --name GOOGLE_CREDENTIALS \
  --value @gcp-project-name-key-id.json
```

## Cluster Credentials

The plugin attempts to fetch credentials for authenticating to the cluster via `kubectl`.

If connecting to a [regional cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/multi-zone-and-regional-clusters#regional), you must provide the `region` parameter to the plugin and omit the `zone` parameter.

If connecting to a [zonal cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/multi-zone-and-regional-clusters#multi-zone), you must provide the `zone` parameter to the plugin and omit the `region` parameter.

The `zone` and `region` parameters are mutually exclusive; providing both to the plugin for the same execution will result in an error.

## Using `secrets`

`drone-gke` also supports creating Kubernetes secrets for you. These secrets should be passed from Drone secrets to the plugin as environment variables with targets with the prefix `secret_`. These secrets will be used as variables in the `secret_template` in their environment variable form (uppercased).

Kubernetes expects secrets to be base64 encoded, `drone-gke` does that for you. If you pass in a secret that is already base64 encoded, please apply the prefix `secret_base64_` and the plugin will not re-encode them.

## Available vars

These variables are always available to reference in any manifest, and cannot be overwritten by `vars` or `secrets`:

```json
{
  "BRANCH": "main",
  "BUILD_NUMBER": "12",
  "COMMIT": "4923x0c3380413ec9288e3c0bfbf534b0f18fed1",
  "TAG": "",
  "cluster": "my-gke-cluster",
  "namespace": "",
  "project": "my-gcp-proj",
  "zone": "us-east1-a"
}
```

## Expanding environment variables

It may be desired to reference an environment variable for use in the Kubernetes manifest.
In order to do so in `vars`, the `expand_env_vars` must be set to `true`.

For example when using `drone deploy org/repo 5 production -p IMAGE_VERSION=1.0`, to get `IMAGE_VERSION` in `vars`:

```yml
expand_env_vars: true
vars:
  image: my-image:$${IMAGE_VERSION}
```

The equivalent command for Drone 1.0 and above would be: `drone build promote org/repo 5 production -p IMAGE_VERSION=1.0`

The plugin will [expand the environment variable][expand] for the template variable.

To use `$${IMAGE_VERSION}` or `$IMAGE_VERSION`, see the [Drone docs][environment] about preprocessing.
`${IMAGE_VERSION}` will be preprocessed to an empty string.

[expand]: https://golang.org/pkg/os/#ExpandEnv
[environment]: http://docs.drone.io/environment/

## Using "extra" `kubectl` versions

### tl;dr

To run `drone-gke` using a different version of `kubectl` than the default, set `kubectl-version` to the version you'd like to use.

For example, to use the **1.14** version of `kubectl`:

```yml
image: nytimes/drone-gke
settings:
  kubectl_version: "1.14"
```

This will configure the plugin to execute `/google-cloud-sdk/bin/kubectl.1.14` instead of `/google-cloud-sdk/bin/kubectl` for all `kubectl` commands.

### Background

Beginning with the [`237.0.0 (2019-03-05)` release of the `gcloud` SDK](https://cloud.google.com/sdk/docs/release-notes#23700_2019-03-05), "extra" `kubectl` versions are installed automatically when `kubectl` is installed via `gcloud components install kubectl`.

These "extra" versions are installed alongside the SDK's default `kubectl` version at `/google-cloud-sdk/bin` and are named using the following pattern:

```sh
kubectl.$clientVersionMajor.$clientVersionMinor
```

To list all of the "extra" `kubectl` versions available within a particular version of `drone-gke`, you can run the following:

```sh
# list "extra" kubectl versions available with nytimes/drone-gke
docker run \
  --rm \
  --interactive \
  --tty \
  --entrypoint '' \
  nytimes/drone-gke list-extra-kubectl-versions
```

## Example reference usage

### `.drone.yml`

Note particularly the `gke:` build step.

```yml
---
kind: pipeline
type: docker
name: default

steps:
  - name: build
    image: golang:1.14
    environment:
      GOPATH: /drone
      CGO_ENABLED: 0
    commands:
      - go get -t
      - go test -v -cover
      - go build -v -a
    when:
      event:
        - push
        - pull_request

  - name: gcr
    image: plugins/gcr
    settings:
      registry: us.gcr.io
      repo: my-gke-project/my-app
      tags:
      - ${DRONE_COMMIT}
    secrets: [google_credentials]
    when:
      event: push
      branch: main

  - name: gke
    image: nytimes/drone-gke
    environment:
      TOKEN:
        from_secret: GOOGLE_CREDENTIALS
      USER: root
      SECRET_API_TOKEN:
        from_secret: APP_API_KEY
      SECRET_BASE64_P12_CERT:
        from_secret: P12_CERT
    settings:
      zone: us-central1-a
      cluster: my-gke-cluster
      namespace: ${DRONE_BRANCH}
      expand_env_vars: true
      vars:
        app: my-app
        env: dev
        image: us.gcr.io/my-gke-project/my-app:${DRONE_COMMIT}
        user: $${USER}
    when:
      event: push
      branch: main
```

### `.kube.yml`

Note the three Kubernetes yml resource manifests separated by `---`.

```yml
---
apiVersion: apps/v1
kind: Deployment

metadata:
  name: {{.app}}-{{.env}}

spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: {{.app}}
        env: {{.env}}
    spec:
      containers:
        - name: app
          image: {{.image}}
          ports:
            - containerPort: 8000
          env:
            - name: APP_NAME
              value: {{.app}}
            - name: USER
              value: {{.user}}
            - name: APP_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{.app}}-{{.env}}
                  key: app-api-key
---
apiVersion: v1
kind: Service

metadata:
  name: {{.app}}-{{.env}}

spec:
  type: NodePort
  selector:
    app: {{.app}}
    env: {{.env}}
  ports:
    - name: http
      protocol: TCP
      port: 80
      targetPort: 8000
---
apiVersion: networking.k8s.io/v1beta1
kind: Ingress

metadata:
  name: {{.app}}-{{.env}}

spec:
  backend:
    serviceName: {{.app}}-{{.env}}
    servicePort: 80
```

### `.kube.sec.yml`

Note that the templated output will not be dumped when debugging.

```yml
---
apiVersion: v1
kind: Secret

metadata:
  name: {{.app}}-{{.env}}

type: Opaque

data:
  app-api-key: {{.SECRET_APP_API_KEY}}
  p12-cert: {{.SECRET_BASE64_P12_CERT}}
```
