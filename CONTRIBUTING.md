# Contributing to drone-gke

`drone-gke` is an open source project started by a handful of developers at The New York Times and open to the entire open source community.

We really appreciate your help!

## Filing issues

When filing an issue, make sure to answer these five questions:

1. What version of Go are you using (`go version`)?
1. What version of Drone are you using?
1. What operating system and processor architecture are you using?
1. What did you do?
1. What did you expect to see?
1. What did you see instead?

Feel free to open issues asking general questions along with feature requests.

## Contributing code

Pull requests are very welcome!
Before submitting changes, please follow these guidelines:

1. Check the open issues and pull requests for existing discussions.
1. Open an issue to discuss a new feature.
1. Write tests.
1. Make sure code follows the ['Go Code Review Comments'](https://github.com/golang/go/wiki/CodeReviewComments).
1. Open a Pull Request.

## Development

### Workflow

The git workflow follows [GitFlow](https://datasift.github.io/gitflow/IntroducingGitFlow.html).

New features should be based on the `master` branch.

### Dependencies

#### Core

```sh
# run tests for required core dependencies
make check
```

#### Go

[Go Modules](https://blog.golang.org/using-go-modules) is used to manage dependencies.

### Build, Test, Push

#### Building the `drone-gke` executable

```sh
make drone-gke
```

#### Testing the `drone-gke` executable

```sh
make test-coverage
```

#### Building the `drone-gke` docker image

Using a custom repo name:

```sh
docker_repo_name=alice make docker-image
```

Using a custom tag:

```sh
docker_tag=alpha make docker-image
```

#### Pushing `drone-gke` docker images

> :warning: By default, `make docker-push` will attempt to push to [`nytimes/drone-gke`](https://hub.docker.com/r/nytimes/drone-gke)

Push existing image to custom repo using custom tag:

```sh
docker_repo_name=alice docker_tag=alpha make docker-push
```

### Running `drone-gke` locally

> :warning: Valid GCP service account credentials are required. See ["Creating test service account credentials"](#creating-test-service-account-credentials) for more details.

#### Configuring [`TOKEN`](DOCS.md#secrets)

If you created test service account credentials using ["Creating test service account credentials"](#creating-test-service-account-credentials), those credentials will be used by default.

If using service account credentials that you created manually:

```sh
# replace with correct path
export TOKEN="$(cat /path/to/keyfile.json)"
```

#### Configuring cluster related values

If you have an existing `gcloud` configuration for `container/cluster`, `compute/zone`, or `compute/region`, those values will be used by default.

Otherwise manually configure the following using appropriate values:

```sh
export PLUGIN_CLUSTER=dev
export PLUGIN_ZONE=us-east1-b
# -- or --
export PLUGIN_REGION=us-east1
```

By default, `PLUGIN_NAMESPACE` will be set to `drone-gke-test`.

If you'd like to use the namespace currently configured with `kubectl`:

```sh
export PLUGIN_NAMESPACE=$(kubectl config view --minify --output 'jsonpath={..namespace}')
```

Otherwise manually configure a different namespace value:

```sh
export PLUGIN_NAMESPACE=my-test-ns
```

#### Running a `drone-gke` docker image

> :warning: Assumes `TOKEN` and any cluster related parameters have already been set / exported within the current terminal session

Using sample configurations within `local-example`:

```sh
# `dry-run` and `verbose` are both enabled by default
make run

# disable via the respective `PLUGIN_*` environment variables
PLUGIN_DRY_RUN=0 PLUGIN_VERBOSE=0 make run
```

Using custom configurations with default files:

```sh
# Deploy manifest templates in my-custom-configs

# my-custom-configs
# ├── .kube.sec.yml
# ├── .kube.yml
# └── vars.json

export CONFIG_HOME=$(pwd)/my-custom-configs
make run
```

Using custom configurations with _custom_ files:

```sh
# Deploy manifest templates in my-super-custom-configs

# my-super-custom-configs
# ├── app.yml
# ├── custom-vars.json
# └── secrets.yml

export CONFIG_HOME=$(pwd)/my-super-custom-configs
export PLUGIN_SECRET_TEMPLATE=${CONFIG_HOME}/secrets.yml
export PLUGIN_TEMPLATE=${CONFIG_HOME}/app.yml
export PLUGIN_VARS="$(cat ${CONFIG_HOME}/custom-vars.json)"
make run
```

Using custom kubectl version:

```sh
export PLUGIN_KUBECTL_VERSION=1.14
make run
```

If you've built the docker image using a custom repo or tag:

```sh
export docker_repo_name=alice
export docker_tag=alpha
make run
```

To print the commands that would be executed, without _actually_ executing them:

```sh
make run -n
```

### Tips and Tricks

#### `make e2e`

`make e2e` is an alias for:

```sh
make drone-gke
make test-coverage
make docker-image
make test-resources
make run
```

Build and test the `drone-gke` binary, build a `drone-gke` docker image tagged using a custom repo name, and run that docker image against a custom test cluster:

```sh
export docker_repo_name=alice
export PLUGIN_CLUSTER=my-test-cluster
make e2e
```

#### `make release`

> :warning: By default, `make release` will attempt to push to [`nytimes/drone-gke`](https://hub.docker.com/r/nytimes/drone-gke)

`make release` is an alias for:

```sh
make drone-gke
make test-coverage
make docker-image
make docker-push
```

Build, test, and push new image using custom repo, tag values:

```sh
export docker_repo_name=alice
export docker_tag=alpha
make release
```

#### Creating test service account credentials

> :warning: The following requires access to an active GCP project, a GCP user account with sufficient privileges, properly configured `gcloud`, and `terraform` executables. Please review ["Creating and managing service accounts"](https://cloud.google.com/iam/docs/creating-managing-service-accounts) and the Terraform ["Google Cloud Platform Provider"](https://www.terraform.io/docs/providers/google/index.html) documentation before continuing.

##### Setup

Verify `gcloud` configuration:

```sh
# the current / active GCP project will be the owner of your test service account resources
gcloud config list
```

##### Creation and configuration

Run following to verify the test resources that will be created / configured:

```sh
make terraform/test.tfplan
```

Run the following to create / configure the test resources:

```sh
make test-resources
```

##### Clean up

Once you've finished testing, don't forget to delete these test resources:

```sh
make destroy-test-resources
```

## License

Unless otherwise noted, `drone-gke` is distributed under the Apache 2.0-style license found in [LICENSE](LICENSE).
