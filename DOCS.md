**This is currently very new and unstable.**

Please don't use it for anything important!

Use this plugin to deploy Docker images to Google Container Engine (GKE).
Please read the GKE [documentation](https://cloud.google.com/container-engine/) before you begin.
You will need to generate a service account and use it's [JSON credential file](https://cloud.google.com/storage/docs/authentication#service_accounts) to authenticate.

## Overview

The project is inferred from the JSON credentials.

The following parameters are used to configure this plugin:

* `image` - this plugin's Docker image
* `zone` - zone of the container cluster
* `cluster` - name of the container cluster
* `token` - service account's JSON credentials
* *optional* `template` - Kubernetes template (like the [deployment object](http://kubernetes.io/docs/user-guide/deployments/)) (defaults to `.kube.yml`)
* *optional* `secret_template` - Kubernetes template for the [secret object](http://kubernetes.io/docs/user-guide/secrets/) (defaults to `.kube.sec.yml`)
* `vars` - variables to use in `template`
* `secrets` - variables to use in `secret_template`

Optional (useful for debugging):

* `dry_run` - do not apply the Kubernetes templates (defaults to `false`)
* `verbose` - dump available `vars` and the generated Kubernetes `template` (excluding secrets) (defaults to `false`)

## Templates

For details about the JSON Token, please view the [drone-gcr plugin](https://github.com/drone-plugins/drone-gcr/blob/master/DOCS.md#json-token).

## Examples

`.drone.yml`, particularly the `deploy:` section:
```yml
build:
  image: golang:1.7
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
    repo: project-1/my-app
    tag: "$$COMMIT"
    token: >
      $$GOOGLE_CREDENTIALS

    when:
      event: push
      branch: master

deploy:
  gke:
    image: nytimes/drone-gke

    zone: us-central1-a
    cluster: my-k8s-cluster
    token: >
      $$GOOGLE_CREDENTIALS

    vars:
      image: gcr.io/project-1/my-app:$$COMMIT
      app: my-app
      env: dev
      NAME: example
    secrets:
      github_token: $$GITHUB_TOKEN

    when:
      event: push
      branch: master
```

Example `.kube.yml`, notice the two yml configs separated by `---`:
```yml
apiVersion: extensions/v1beta1
kind: Deployment

metadata:
  name: {{.app}}-{{.env}}-deployment

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
            - name: NAME
              value: {{.NAME}}
            - name: GITHUB_TOKEN
              valueFrom:
                secretKeyRef:
                  name: secrets
                  key: github-token
---
apiVersion: v1
kind: Service

metadata:
  name: {{.app}}-{{.env}}-svc

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
apiVersion: v1
kind: Secret

metadata:
  name: secrets

type: Opaque

data:
  github-token: {{.GITHUB_TOKEN}}
```
