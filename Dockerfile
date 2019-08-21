# see https://hub.docker.com/r/google/cloud-sdk/tags for available tags / versions
ARG GCLOUD_SDK_TAG

FROM google/cloud-sdk:${GCLOUD_SDK_TAG}

ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true
ENV CLOUDSDK_CORE_DISABLE_PROMPTS=1

RUN \
  gcloud --no-user-output-enabled components install kubectl && \
    rm -rf /google-cloud-sdk/.install

ADD drone-gke bin/set-env-versions bin/list-extra-kubectl-versions /usr/local/bin/

ENTRYPOINT ["set-env-versions", "drone-gke"]
