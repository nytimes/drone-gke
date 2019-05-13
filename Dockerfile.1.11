# kubectl 1.11.9
# https://cloud.google.com/sdk/docs/release-notes
FROM google/cloud-sdk:241.0.0-alpine

# Install kubectl
RUN gcloud components install kubectl && \
    rm -rf ./google-cloud-sdk/.install

ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true

# Add the Drone plugin
ADD drone-gke /bin/

ENTRYPOINT ["/bin/drone-gke"]
