# drone-gke

Drone plugin to deploy images to Kubernetes on Google Container Engine.
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

This is a little simpler than deploying straight to Kube, because the api endpoints and credentials and whatnot can be derived using the Google credentials.

## Development

[`glide`](https://github.com/Masterminds/glide) is used to manage vendor dependencies.

```bash
go build
```

## See also

* [UKHomeOffice/drone-kubernetes](https://github.com/UKHomeOffice/drone-kubernetes)
