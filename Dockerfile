# kubectl 1.6.6
#FROM google/cloud-sdk:161.0.0-alpine

# kubectl 1.7.0
FROM google/cloud-sdk:181.0.0-alpine

# Install kubectl
RUN gcloud components install kubectl

ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true

# Add the Drone plugin
ADD drone-gke /bin/

ENTRYPOINT ["/bin/drone-gke"]
