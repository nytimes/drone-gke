# drone-gke

Drone plugin to deploy container images to Kubernetes on Google Container Engine.
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

Simplify deploying to Google Kubernetes Engine.
Derive the API endpoints and credentials from the Google credentials and open the yaml file to templatization and customization with each Drone build.

## Links

- Usage [documentation](DOCS.md)
- Docker Hub [release tags](https://hub.docker.com/r/nytimes/drone-gke/tags)
- Drone.io [builds](https://beta.drone.io/nytimes/drone-gke)

## Releases and versioning

### Tool

Use the `x.X` releases for stable use cases (eg 0.8).
Breaking changes may occur between `x.X` releases (eg 0.7 and 0.8), and will be documented in the [release notes](https://github.com/nytimes/drone-gke/releases).

### Kubernetes API

Use the release [tag](https://hub.docker.com/r/nytimes/drone-gke/tags/) suffixed with your desired `kubectl` version.
The last two-three minor releases are supported ([same as GKE](https://cloud.google.com/kubernetes-engine/versioning-and-upgrades)).

### Container

- Pushes to the [`develop`](https://github.com/nytimes/drone-gke/tree/develop) branch will update the image tagged `develop`.
- Pushes to the [`master`](https://github.com/nytimes/drone-gke/tree/master) branch will update the images tagged `latest` and corresponding `kubectl` versions.
- Tags to the [`master`](https://github.com/nytimes/drone-gke/tree/master) branch will create the images with the tag value (eg `0.7.1` and `0.7`) and corresponding `kubectl` versions.

## Development

The git workflow follows git-flow.
New features should be based on the `master` branch.

Go Modules is used to manage dependencies.

### Go

#### Building drone-gke executable

```bash
make drone-gke
```

#### Testing the executable

```bash
go test
```

### Docker

#### Building images

Using a custom repo name:

```sh
docker_repo_name=your-docker-hub-user make docker-build
```

Using a custom tag:

```sh
docker_tag=alpha make docker-build
```

#### Publishing images

Existing images:

```sh
# runs docker push with custom repo, tag values
docker_repo_name=your-docker-hub-user docker_tag=alpha make docker-release
```

## All-in-one

```sh
# runs go build, docker build, docker push using defaults
make dist
```

## Usage

### Running within a Drone pipeline

Please take a look at [the docs](DOCS.md).

### Running locally

Using sample configurations within `local-example`:

```sh
# Deploy the manifest templates in local-example/
export GOOGLE_APPLICATION_CREDENTIALS=~/tmp/key.json
export PLUGIN_CLUSTER=dev
export PLUGIN_NAMESPACE=drone-gke-test
export PLUGIN_ZONE=us-east1-b
export docker_cmd='--verbose --dry-run' # Remove --dry-run to deploy
make docker-run-local
```

Using custom configurations with default file names:

```sh
# Deploy manifest templates in my-custom-configs

# my-custom-configs
# ├── .kube.sec.yml
# ├── .kube.yml
# └── vars.json

export CONFIG_HOME=$(pwd)/my-custom-configs
make docker-run-local
```

Using custom configurations with custom file names:

```sh
# Deploy manifest templates in my-super-custom-configs

# my-super-custom-configs
# ├── app.yml
# ├── custom-vars.json
# └── secrets.yml

export CONFIG_HOME=$(pwd)/my-super-custom-configs
export PLUGIN_SECRET_TEMPLATE=${CONFIG_HOME}/secrets.yml
export PLUGIN_TEMPLATE=${CONFIG_HOME}/app.yml
export PLUGIN_VARS_PATH=${CONFIG_HOME}/custom-vars.json
make docker-run-local
```

If you have an existing `gcloud` configuration for `container/cluster`:

```sh
export PLUGIN_CLUSTER=$(gcloud --quiet config get-value container/cluster 2>/dev/null)
make docker-run-local
```

If your current `kubectl` context is set for a particular namespace:

```sh
export PLUGIN_NAMESPACE=$(kubectl config view --minify --output 'jsonpath={..namespace}')
make docker-run-local
```

If you have an existing `gcloud` configuration for `compute/zone`:

```sh
export PLUGIN_ZONE=$(gcloud --quiet config get-value compute/zone 2>/dev/null)
make docker-run-local
```

If you've built the docker image using a custom repo or tag:

```sh
export docker_repo_name=nytm
export docker_tag=alpha
make docker-run-local
```

To print the commands that would be executed, without _actually_ executing them:

```sh
make docker-run-local -n
```
