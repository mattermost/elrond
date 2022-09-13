# Build the Mattermost Elrond
ARG DOCKER_BUILD_IMAGE=golang:1.18.1
ARG DOCKER_BASE_IMAGE=alpine:3.16.2

FROM ${DOCKER_BUILD_IMAGE} AS build
WORKDIR /elrond/
COPY . /elrond/
RUN apt-get update -yq && apt-get install -yq unzip
RUN make build


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

RUN apk add curl 

ENV DOCKERVERSION=20.10.17
RUN curl -fsSLO https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKERVERSION}.tgz \
  && tar xzvf docker-${DOCKERVERSION}.tgz --strip 1 \
                 -C /usr/local/bin docker/docker \
  && rm docker-${DOCKERVERSION}.tgz

RUN  apk update && apk add libc6-compat && apk add ca-certificates
COPY --from=build /elrond/build/_output/bin/elrond /elrond/elrond
COPY --from=build /elrond/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup
WORKDIR /elrond/

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
