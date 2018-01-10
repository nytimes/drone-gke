FROM google/cloud-sdk:latest

# Install kubectl
RUN apt-get install kubectl

# Add the Drone plugin
ADD drone-gke /bin/

ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true
ENTRYPOINT ["/bin/drone-gke"]
