> ⚠️  This branch contains experimental changes and should not be used in production

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

## See also

* [UKHomeOffice/drone-kubernetes](https://github.com/UKHomeOffice/drone-kubernetes)
