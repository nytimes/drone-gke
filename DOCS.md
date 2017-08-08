Use this plugin to deploy Docker images to [Google Container Engine (GKE)][gke].

[gke]: https://cloud.google.com/container-engine/

## Overview

The following parameters are used to configure this plugin:

* `image` - this plugin's Docker image
* `zone` - zone of the container cluster
* `cluster` - name of the container cluster
* *optional* `namespace` - Kubernetes namespace to operate in (defaults to `default`)
* *optional* `template` - Kubernetes manifest template (defaults to `.kube.yml`)
* *optional* `secret_template` - Kubernetes [_Secret_ resource](http://kubernetes.io/docs/user-guide/secrets/) manifest template (defaults to `.kube.sec.yml`)
* `vars` - variables to use in `template`

### Debugging parameters

These optional parameters are useful for debugging:

* `dry_run` - do not apply the Kubernetes templates (defaults to `false`)
* `verbose` - dump available `vars` and the generated Kubernetes `template` (excluding secrets) (defaults to `false`)

## Credentials

`drone-gke` requires a Google service account and use it's [JSON credential file][service-account] to authenticate.

This must be passed to the plugin under the target `token`.

The plugin infers the GCP project from the JSON credentials (`token`) and retrieves the GKE cluster credentials.

[service-account]: https://cloud.google.com/storage/docs/authentication#service_accounts

### Setting the JSON token

Improved in Drone 0.5+, it is no longer necessary to align the JSON file.

#### GUI

Simply copy the contents of the JSON credentials file and paste it in the input field (for example for a secret named `GOOGLE_CREDENTIALS`).

##### CLI

```
drone secret add \
--event push \
--event pull_request \
--event tag \
--event deployment \
--repository nytm/dv-cachet \
--name GOOGLE_CREDENTIALS \
--value @gcp-project-name-key-id.json
```

## Secrets

`drone-gke` also supports creating Kubernetes secrets for you. These secrets should be passed from Drone secrets to the plugin as environment variables with targets with the prefix `secret_`. These secrets will be used as variables in the `secret_template` in their environment variable form (uppercased).

Kubernetes expects secrets to be base64 encoded, `drone-gke` does that for you. If you pass in a secret that is already base64 encoded, please apply the prefix `secret_base64_` and the plugin will not re-encode them.

## Example reference usage

### `.drone.yml`

Note particularly the `gke:` build step.

```yml
---
pipeline:
  build:
    image: golang:1.8
    environment:
      - GOPATH=/drone
      - CGO_ENABLED=0
    commands:
      - go get -t
      - go test -v -cover
      - go build -v -a
    when:
      event:
        - push
        - pull_request

  gcr:
    storage_driver: overlay
    repo: my-gke-project/my-app
    tag: ${DRONE_COMMIT}
    secrets:
      - source: GOOGLE_CREDENTIALS
        target: token
    when:
      event: push
      branch: master

  gke:
    image: nytimes/drone-gke
    zone: us-central1-a
    cluster: my-k8s-cluster
    namespace: ${DRONE_BRANCH}
    vars:
      image: gcr.io/my-gke-project/my-app:${DRONE_COMMIT}
      app: my-app
      env: dev
    secrets:
      - source: GOOGLE_CREDENTIALS
        target: token
      - source: API_TOKEN
        target: secret_api_token
      - source: P12_CERT
        target: secret_base64_p12_cert
    when:
      event: push
      branch: master
```

### `.kube.yml`

Note the two Kubernetes yml resource manifests separated by `---`.

```yml
---
kind: Deployment
apiVersion: extensions/v1beta1

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
            - name: API_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{.app}}-{{.env}}
                  key: api-token
---
kind: Service
apiVersion: v1

metadata:
  name: {{.app}}-{{.env}}

spec:
  type: LoadBalancer
  selector:
    app: {{.app}}
    env: {{.env}}
  ports:
    - port: 80
      targetPort: 8000
      protocol: TCP
```

### `.kube.sec.yml`

Note that the templated output will not be dumped when debugging.

```yml
kind: Secret
apiVersion: v1

metadata:
  name: {{.app}}-{{.env}}

type: Opaque

data:
  api-token: {{.SECRET_API_TOKEN}}
  p12-cert: {{.SECRET_BASE64_P12_CERT}}
```
