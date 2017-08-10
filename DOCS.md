Use this plugin to deploy Docker images to Google Container Engine (GKE).
Please read the GKE [documentation](https://cloud.google.com/container-engine/) before you begin.
You will need to generate a service account and use it's [JSON credential file](https://cloud.google.com/storage/docs/authentication#service_accounts) to authenticate.

## Overview

The project is inferred from the JSON credentials (`token`).

The following parameters are used to configure this plugin:

* `image` - this plugin's Docker image
* `zone` - zone of the container cluster
* `cluster` - name of the container cluster
* `namespace` - Kubernetes namespace to operate in
* `token` - service account's JSON credentials
* *optional* `template` - Kubernetes template (like the [deployment object](http://kubernetes.io/docs/user-guide/deployments/)) (defaults to `.kube.yml`)
* *optional* `secret_template` - Kubernetes template for the [secret object](http://kubernetes.io/docs/user-guide/secrets/) (defaults to `.kube.sec.yml`)
* `vars` - variables to use in `template`

### Optional

These optional parameters are useful for debugging:

* `dry_run` - do not apply the Kubernetes templates (defaults to `false`)
* `verbose` - dump available `vars` and the generated Kubernetes `template` (excluding secrets) (defaults to `false`)

## Secrets

`drone-gke` requires a google service account key in json format to run, and this must be passed to the plugin under the target `token`.

`drone-gke` also supports creating kubernetes secrets for you. These secrets should be passed with targets with the prefix `secret_`. These secrets will be used as variables in the `secret_template`.

Kubernetes expects secrets to be base64 encoded, `drone-gke` does that for you. If you pass in a secret that is already base64 encoded, please apply the prefix `secret_base64_` and the plugin will not re-encode them.

## Templates

For details about the JSON Token, please view the [drone-gcr plugin](https://github.com/drone-plugins/drone-gcr/blob/master/DOCS.md#json-token).

## Examples

`.drone.yml`, particularly the `deploy:` section:
```yml
build:
  image: golang:1.7

  environment:
    - GOPATH=/drone

  commands:
    - go get -t
    - go test -v -cover
    - CGO_ENABLED=0 go build -v -a

  when:
    event:
      - push
      - pull_request

publish:
  gcr:
    storage_driver: overlay
    repo: my-gke-project/my-app
    tag: "$$COMMIT"

    secrets:
    - source: GOOGLE_CREDENTIALS
      target: token

    when:
      event: push
      branch: master

deploy:
  gke:
    image: nytimes/drone-gke

    zone: us-central1-a
    cluster: my-k8s-cluster
    namespace: $$BRANCH

    vars:
      image: gcr.io/my-gke-project/my-app:$$COMMIT
      app: my-app
      env: dev
    secrets:
      api_token: $$API_TOKEN
    secrets_base64:
      p12_cert: $$P12_CERT
    

    secrets:
    - source: GOOGLE_CREDENTIALS
      target: token
    - source: API_TOKEN
      target: secret_api_token
    - source: P12_CERT
      target: p12_cert

    when:
      event: push
      branch: master
```

Example `.kube.yml`, notice the two yml configs separated by `---`:
```yml
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

## JSON Token

See documentation from the [drone-gcr][drone-gcr] plugin on setting the JSON token.

[drone-gcr]: https://github.com/drone-plugins/drone-gcr/blob/master/DOCS.md#json-token
