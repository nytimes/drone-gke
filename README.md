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

Users should use the `x.X` releases for stable use cases (eg 0.8).

Breaking changes may occur between `x.X` releases (eg 0.7 and 0.8), and will be documented in the [release notes](https://github.com/nytimes/drone-gke/releases).

Use the release [tag](https://hub.docker.com/r/nytimes/drone-gke/tags/) suffixed with your desired `kubectl` version.
The last two-three minor releases are supported ([same as GKE](https://cloud.google.com/kubernetes-engine/versioning-and-upgrades)).

- Pushes to the [`develop`](https://github.com/NYTimes/drone-gke/tree/develop) branch will update the image tagged `develop`.
- Pushes to the [`master`](https://github.com/NYTimes/drone-gke/tree/master) branch will update the images tagged `latest` and corresponding `kubectl` versions.
- Tags to the [`master`](https://github.com/NYTimes/drone-gke/tree/master) branch will create the images with the tag value (eg `0.7.1` and `0.7`) and corresponding `kubectl` versions.

## Development

The git workflow follows git-flow.
New features should be based on the `master` branch.

[`glide`](https://github.com/Masterminds/glide) is used to manage vendor dependencies.

```bash
go build
```

## Testing

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
