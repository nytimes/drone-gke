FROM google/cloud-sdk:183.0.0-alpine

# Install kubectl
RUN gcloud components install kubectl -q

# Add the Drone plugin
ADD drone-gke /bin/

ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true
ENTRYPOINT ["/bin/drone-gke"]
