# Build the Mattermost Elrond
ARG DOCKER_BUILD_IMAGE=golang:1.20
ARG DOCKER_BASE_IMAGE=alpine:3.19

FROM --platform=${TARGETPLATFORM} ${DOCKER_BUILD_IMAGE} AS build
ARG TARGETARCH
WORKDIR /elrond/
COPY . /elrond/
ENV ARCH=${TARGETARCH}

RUN make build ARCH=${ARCH}
RUN apt-get update -yq && apt-get install -yq unzip


# Final Image
FROM --platform=${TARGETPLATFORM} ${DOCKER_BASE_IMAGE}
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
COPY --from=build /elrond/build/_output/bin/elrond /elrond/elrond
COPY --from=build /elrond/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup
WORKDIR /elrond/

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
