# drone-gke

Drone plugin to deploy container images to Kubernetes on Google Container Engine.
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

This is a little simpler than deploying straight to Kubernetes, because the API endpoints and credentials can be derived using the Google credentials.
In addition, this opens the yaml file to templatization and customization with each Drone build.

## Drone compatibility

For usage in Drone 0.5 and newer, please use a [release tag](https://hub.docker.com/r/nytimes/drone-gke/tags/) >= `0.7`.

For usage in Drone 0.4, please use a `0.4` [release tag](https://hub.docker.com/r/nytimes/drone-gke/tags/).

## Releases

Users should use the `x.X` releases for stable use cases (eg 0.8).

Breaking changes may occur between `x.X` releases (eg 0.7 and 0.8), and will be documented in the release notes.

- Pushes to the [`develop`](https://github.com/NYTimes/drone-gke/tree/develop) branch will update the Docker Hub release tagged `develop`.
- Pushes to the [`master`](https://github.com/NYTimes/drone-gke/tree/master) branch will update the Docker Hub release tagged `latest`.
- Tags to the [`master`](https://github.com/NYTimes/drone-gke/tree/master) branch will create the Docker Hub release with the tag value (eg `0.7.1` and `0.7`).

## kubectl version

Use the [release tag](https://hub.docker.com/r/nytimes/drone-gke/tags/) suffix with the desired `kubectl` version.
The last three minor releases are supported ([same as GKE](https://cloud.google.com/kubernetes-engine/versioning-and-upgrades)).

## Development

The git workflow follows git-flow. New features should be based on the `master` branch.

[`glide`](https://github.com/Masterminds/glide) is used to manage vendor dependencies.

```bash
go build
```

## Testing

**This could use your contribution!**

```bash
go test
```

## Docker

Build the Docker image with the following commands:

```
OWNER=your-docker-hub-user ./bin/dist
```

## Usage

Using it in a `.drone.yml` pipeline: please take a look at [the docs](DOCS.md).

Executing locally from the working directory:

```
# Deploy the manifest templates in local-example/
$ cd local-example/

# Set to the path of your GCP service account JSON file
$ export GOOGLE_APPLICATION_CREDENTIALS=xxx

# Set to your cluster
$ export CLUSTER=yyy

# Set to your cluster's zone
$ export ZONE=zzz

# The variables required for the templates in JSON format
$ cat vars.json
{
   "app": "echo",
   "env": "dev",
   "image": "gcr.io/google_containers/echoserver:1.4"
}

# Execute the plugin
$ docker run --rm \
  -e PLUGIN_CLUSTER="$CLUSTER" \
  -e PLUGIN_ZONE="$ZONE" \
  -e PLUGIN_NAMESPACE=drone-gke \
  -e PLUGIN_VARS="$(cat vars.json)" \
  -e TOKEN="$(cat $GOOGLE_APPLICATION_CREDENTIALS)" \
  -e SECRET_API_TOKEN=123 \
  -e SECRET_BASE64_P12_CERT="cDEyCg==" \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  nytimes/drone-gke --dry-run --verbose

# Remove --dry-run to deploy
```
