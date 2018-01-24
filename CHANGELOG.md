# Changelog

Deprecated means no longer supported.
Please update to the newest GA release.

## 0.8.0

### FEATURES & IMPROVEMENTS

- Add functionality to wait for successful `Deployment` rollout using `kubectl rollout status`.

## 0.7.1

### FEATURES & IMPROVEMENTS

- `kubectl` is updated to 1.8.4.

## 0.7.0

The plugin is updated to Drone 0.5+ style (environment variables).

### BREAKING CHANGES

- Consumes Drone 0.5+ style secrets, causing breaking changes with secrets and the GCP token across the board.
Please read the documentation for reference usage.

- Reduced available "default" environment variables to the plugin.
To reference other environment variables, pass them via `vars`.
  - `DRONE_BUILD_NUMBER`
  - `DRONE_COMMIT`
  - `DRONE_BRANCH`
  - `DRONE_TAG`

### FEATURES & IMPROVEMENTS

- `kubectl` is updated to 1.6.6.
- Secret templates now have access to non-secret `vars` for template rendering.
- Enforces `secrets` to only be accessible when rendering the `.kube.sec.yml` template to prevent leaking secrets elsewhere.
- The plugin will validate the rendered manifests with `kubectl` before applying them.

## 0.4 (Deprecated)

Stable and works with Drone 0.4.

To continue using this version, update references in `.drone.yml` to pin the plugin to `images: nytimes/drone-gke:0.4`.
