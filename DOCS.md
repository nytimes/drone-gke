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

TODO(tonglil):
Can this token be "string-ified" and pasted in the GUI?
Or must it be uploaded via CLI?
We should document this.

## Secrets

`drone-gke` also supports creating Kubernetes secrets for you. These secrets should be passed with targets with the prefix `secret_`. These secrets will be used as variables in the `secret_template`.

Kubernetes expects secrets to be base64 encoded, `drone-gke` does that for you. If you pass in a secret that is already base64 encoded, please apply the prefix `secret_base64_` and the plugin will not re-encode them.

## Examples

`.drone.yml`, particularly the `gke:` build step:

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
    tag: $DRONE_COMMIT
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
    namespace: $DRONE_BRANCH
    vars:
      image: gcr.io/my-gke-project/my-app:$DRONE_COMMIT
      app: my-app
      env: dev
    secrets:
      - source: GOOGLE_CREDENTIALS
        target: token
      - source: API_TOKEN
        target: api_token
      - source: P12_CERT
        target: p12_cert
    when:
      event: push
      branch: master
```

Example `.kube.yml`, notice the two Kubernetes yml resource manifests separated by `---`:

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
                  name: secrets
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

`.kube.sec.yml`, templated output will not be dumped when debugging:

```yml
kind: Secret
apiVersion: v1

metadata:
  name: secrets

type: Opaque

data:
  api-token: {{.api_token}}
  p12-cert: {{.p12_cert}}
```
