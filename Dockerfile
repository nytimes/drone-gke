# see https://hub.docker.com/r/google/cloud-sdk/tags for available tags / versions
ARG GCLOUD_SDK_VERSION

FROM google/cloud-sdk:${GCLOUD_SDK_VERSION} AS cloud-sdk
ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true
ENV CLOUDSDK_CORE_DISABLE_PROMPTS=1
ADD bin/install-kubectl /usr/local/bin/
RUN install-kubectl && rm -f /usr/local/bin/install-kubectl

FROM cloud-sdk
ADD drone-gke /usr/local/bin/
ENTRYPOINT ["drone-gke"]
