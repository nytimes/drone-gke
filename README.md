# drone-gke

Deploy images to Kubernetes on Google Container Engine. This is a little simpler
than deploying straight to Kube, because the api endpoints and credentials and
whatnot can be derived using the Google credentials.

This is currently very new and unstable.  
Please don't use it for anything important!


## Example

	build:
	  image: golang:1.7
	  commands:
	    - go get -t
	    - go test -v -cover
	    - CGO_ENABLED=0 go build -v -a
	  when:
	    event:
	      - push
	      - pull_request

	publish:
	  gcr:
	    repo: project-1/my-app
	    tag: "$$COMMIT"
	    token: >
	      $$GOOGLE_CREDENTIALS

	    when:
	      event: push
	      branch: master

	deploy:
	  gke:
	  	image: adammck/drone-gke

	    project: project-1
	    zone: us-central1-a
	    cluster: my-k8s-cluster
	    token: >
	      $$GOOGLE_CREDENTIALS

	    template: .kube.yml
	    vars:
	      image: gcr.io/project-1/my-app:$$COMMIT
	      app: my-app
	      env: dev

	    when:
	      event: push
	      branch: master



## See Also

* [UKHomeOffice/drone-kubernetes](https://github.com/UKHomeOffice/drone-kubernetes)


## License

MIT.
