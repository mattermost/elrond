# Build the Mattermost Elrond
ARG DOCKER_BUILD_IMAGE=golang:1.20
ARG DOCKER_BASE_IMAGE=alpine:3.19

FROM ${DOCKER_BUILD_IMAGE} AS build
WORKDIR /elrond/
COPY . /elrond/

# Detect architecture and set ARCH
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH="amd64"; \
    elif [ "$ARCH" = "aarch64" ]; then \
        ARCH="arm64"; \
    elif [ "$ARCH" = "armv7l" ] || [ "$ARCH" = "armv6l" ]; then \
        ARCH="arm"; \
    fi && \
    echo "ARCH=$ARCH" && \
    make build ARCH=$ARCH
RUN apt-get update -yq && apt-get install -yq unzip


# Final Image
FROM ${DOCKER_BASE_IMAGE}
LABEL name="Mattermost Elrond" \
  maintainer="cloud-team@mattermost.com" \
  vendor="Mattermost" \
  distribution-scope="public" \
  url="https://mattermost.com" \
  io.k8s.description="Elrond manages and supports ring-based deployments in Mattermost Cloud" \
  io.k8s.display-name="Mattermost Elrond"

ENV ELROND=/elrond/elrond \
    USER_UID=10001 \
    USER_NAME=elrond

RUN  apk update && apk add libc6-compat && apk add ca-certificates
COPY --from=build /elrond/cmd/elrond /elrond/elrond
COPY --from=build /elrond/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup
WORKDIR /elrond/

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
