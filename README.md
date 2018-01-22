# drone-gke

Drone plugin to deploy container images to Kubernetes on Google Container Engine.
For the usage information and a listing of the available options please take a look at [the docs](DOCS.md).

This is a little simpler than deploying straight to Kubernetes, because the API endpoints and credentials can be derived using the Google credentials.
In addition, this opens the yaml file to templatization and customization with each Drone build.

## Drone Compatibility

For usage in Drone 0.4, please use the `nytimes/drone-gke:0.4` tag.

For usage in Drone 0.5 and newer, please use the `nytimes/drone-gke:0.7` tag.

## Development

[`glide`](https://github.com/Masterminds/glide) is used to manage vendor dependencies.

```bash
go build
```

The git workflow follows git-flow. New features should be based on the `develop` branch.

## Releases

Users should use the `x.X` releases for stable use cases (eg 0.7).

Breaking changes may occur between `x.X` releases (eg 0.7 and 0.8), and will be documented in the changelog.

- Pushes to the [`develop`](https://github.com/NYTimes/drone-gke/tree/develop) branch will update the Docker Hub release tagged `develop`.
- Pushes to the [`master`](https://github.com/NYTimes/drone-gke/tree/master) branch will update the Docker Hub release tagged `latest` and `x.X` (eg 0.7).
- Tags to the [`master`](https://github.com/NYTimes/drone-gke/tree/master) branch will create the Docker Hub release with the tag value (eg 0.7.1).

## Testing

**This could use your contribution!**
Help us create a runnable test suite.

## See also

* [UKHomeOffice/drone-kubernetes](https://github.com/UKHomeOffice/drone-kubernetes)
