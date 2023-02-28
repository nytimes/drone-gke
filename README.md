# drone-gke

[![Build Status](https://github.com/nytimes/drone-gke/actions/workflows/build.yml/badge.svg)](https://github.com/nytimes/drone-gke/actions/workflows/build.yml)

Drone plugin to deploy container images to Kubernetes on Google Container Engine.
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

Simplify deploying to Google Kubernetes Engine.
Derive the API endpoints and credentials from the Google credentials and open the yaml file to templatization and customization with each Drone build.

## Links

- Usage [documentation](DOCS.md)
- Docker Hub [release tags](https://hub.docker.com/r/nytimes/drone-gke/tags)
- GitHub Actions Workflow [runs](https://github.com/nytimes/drone-gke/actions)
- Contributing [documentation](.github/CONTRIBUTING.md)

## Releases and versioning

### Tool

This tool follows [semantic versioning](https://semver.org/).

Use the minor version (`x.X`) releases for stable use cases (eg 0.9).
Changes are documented in the [release notes](https://github.com/nytimes/drone-gke/releases).

- Pushes to the [`main`](https://github.com/nytimes/drone-gke/tree/main) branch will update the image tagged `latest`.
- Releases will create the images with each major/minor/patch tag values (eg `0.7.1` and `0.7`).

### Kubernetes API

Since the [237.0.0 (2019-03-05) Google Cloud SDK][sdk], the container image contains multiple versions of `kubectl`.
The corresponding client version that matches the cluster version will be used automatically.
This follows the minor release support that [GKE offers](https://cloud.google.com/kubernetes-engine/versioning-and-upgrades).

If you want to use a different version, you can specify the version of `kubectl` used with the [`kubectl_version` parameter][version-parameter].

[sdk]: https://cloud.google.com/sdk/docs/release-notes#23700_2019-03-05
[version-parameter]: DOCS.md#kubectl_version

## Usage

> :warning: For usage within in a `.drone.yml` pipeline, please take a look at [the docs](DOCS.md)

Executing locally from the working directory:

```sh
# Deploy the manifest templates in local-example/
cd local-example/

# Set to the path of your GCP service account JSON-formatted key file
export JSON_TOKEN_FILE=xxx

# Set to your cluster
export PLUGIN_CLUSTER=yyy

# Set to your cluster's zone
export PLUGIN_ZONE=zzz

# Set to a namespace within your cluster's
export PLUGIN_NAMESPACE=drone-gke

# Example variables referenced within .kube.yml
export PLUGIN_VARS="$(cat vars.json)"
# {
#   "app": "echo",
#   "env": "dev",
#   "image": "gcr.io/google_containers/echoserver:1.4"
# }

# Example secrets referenced within .kube.sec.yml
export SECRET_APP_API_KEY=123
export SECRET_BASE64_P12_CERT="cDEyCg=="

# Execute the plugin
docker run --rm \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  -e PLUGIN_TOKEN="$(cat $JSON_TOKEN_FILE)" \
  -e PLUGIN_CLUSTER \
  -e PLUGIN_ZONE \
  -e PLUGIN_NAMESPACE \
  -e PLUGIN_VARS \
  -e SECRET_APP_API_KEY \
  -e SECRET_BASE64_P12_CERT \
  nytimes/drone-gke --dry-run --verbose

# Remove --dry-run to deploy
```
