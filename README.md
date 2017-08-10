# drone-gke

Drone plugin to deploy container images to Kubernetes on Google Container Engine.
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

This is a little simpler than deploying straight to Kubernetes, because the API endpoints and credentials can be derived using the Google credentials.
In addition, this opens the yaml file to templatization and customization with each Drone build.

## Development

[`glide`](https://github.com/Masterminds/glide) is used to manage vendor dependencies.

```bash
go build
```

## Testing

**This could use your contribution!**
Help us create a runnable test suite.

## Docker

Build the Docker image with the following commands:

```
OWNER=your-docker-hub-user ./bin/dist
```

## Usage

Executing locally from the working directory:

```
# The variables required for the templates in JSON format
$ cat vars.json
{
   "app" : "my-app",
   "image" : "gcr.io/my-gke-project/my-app:d8dbe4d94f15fe89232e0402c6e8a0ddf21af3ab",
   "env" : "dev"
}

# Execute the plugin
$ docker run --rm \
  -e PLUGIN_ZONE=us-central1-a \
  -e PLUGIN_CLUSTER=my-gke-cluster \
  -e PLUGIN_NAMESPACE=my-branch \
  -e PLUGIN_VARS="$(cat vars.json)" \
  -e TOKEN="$(cat my-service-account-credential.json)" \
  -e SECRET_API_TOKEN=123 \
  -e SECRET_BASE64_P12_CERT="cDEyCg==" \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  nytimes/drone-gke --dry-run --verbose
```
