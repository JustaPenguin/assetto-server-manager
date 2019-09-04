FROM golang:1.13 AS build

ARG SM_VERSION
ENV DEBIAN_FRONTEND noninteractive
ENV BUILD_DIR ${GOPATH}/src/github.com/cj123/assetto-server-manager
ENV GO111MODULE on

RUN curl -sL https://deb.nodesource.com/setup_12.x | bash -
RUN apt-get update && apt-get install -y build-essential libssl-dev curl nodejs tofrodos dos2unix zip

ADD . ${BUILD_DIR}
WORKDIR ${BUILD_DIR}
RUN rm -rf cmd/server-manager/typescript/node_modules
RUN VERSION=${SM_VERSION} make deploy
RUN mv cmd/server-manager/build/linux/server-manager /usr/bin/

FROM debian:stable-slim AS run
MAINTAINER Callum Jones <cj@icj.me>

ENV DEBIAN_FRONTEND noninteractive

ENV SERVER_USER assetto
ENV SERVER_MANAGER_DIR /home/${SERVER_USER}/server-manager/
ENV SERVER_INSTALL_DIR ${SERVER_MANAGER_DIR}/assetto

# dependencies for plugins, e.g. stracker, kissmyrank
RUN apt-get update && apt-get install -y lib32gcc1 lib32stdc++6 zlib1g zlib1g lib32z1 ca-certificates && rm -rf /var/lib/apt/lists/*

RUN useradd -ms /bin/bash ${SERVER_USER}

RUN mkdir -p ${SERVER_MANAGER_DIR} && mkdir ${SERVER_INSTALL_DIR}

RUN chown -R ${SERVER_USER}:${SERVER_USER} ${SERVER_MANAGER_DIR}
RUN chown -R ${SERVER_USER}:${SERVER_USER} ${SERVER_INSTALL_DIR}

COPY --from=build /usr/bin/server-manager /usr/bin/

USER ${SERVER_USER}
WORKDIR ${SERVER_MANAGER_DIR}

# recommend volume mounting the entire assetto corsa directory
VOLUME ["${SERVER_INSTALL_DIR}"]
EXPOSE 8772
EXPOSE 9600
EXPOSE 8081

ENTRYPOINT ["server-manager"]