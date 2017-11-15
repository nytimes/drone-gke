# kubectl 1.8.3
FROM google/cloud-sdk:180.0.0-alpine

# Install kubectl
RUN gcloud components install kubectl

ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true

# Add the Drone plugin
ADD drone-gke /bin/

ENTRYPOINT ["/bin/drone-gke"]
