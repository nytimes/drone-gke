FROM alpine:3.4

RUN apk add --no-cache curl python

ENV GOOGLE_CLOUD_SDK_VERSION=161.0.0

# Install the gcloud SDK
RUN curl -fsSLo google-cloud-sdk.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-$GOOGLE_CLOUD_SDK_VERSION-linux-x86_64.tar.gz
RUN tar -xzf google-cloud-sdk.tar.gz
RUN rm google-cloud-sdk.tar.gz
RUN ./google-cloud-sdk/install.sh --quiet

# Install kubectl
RUN ./google-cloud-sdk/bin/gcloud components install kubectl

# Install istioctl
ENV ISTIOCTL_VERSION=0.4.0
RUN \
  curl -fsSLo istio.tar.gz https://github.com/istio/istio/releases/download/$ISTIOCTL_VERSION/istio-$ISTIOCTL_VERSION-linux.tar.gz \
  && tar -xzf istio.tar.gz \
  && mv istio-$ISTIOCTL_VERSION istio \
  && rm istio.tar.gz

ENV CLOUDSDK_CONTAINER_USE_APPLICATION_DEFAULT_CREDENTIALS=true

# Clean up
RUN rm -rf ./google-cloud-sdk/.install

# Add the Drone plugin
ADD drone-gke /bin/

ENTRYPOINT ["/bin/drone-gke"]
