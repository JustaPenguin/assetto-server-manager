FROM node:14-buster AS build

RUN curl -fsSL https://go.dev/dl/go1.15.15.linux-amd64.tar.gz | tar -C /usr/ -xzf -
ENV GOROOT=/usr/go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

ARG SM_VERSION
ENV DEBIAN_FRONTEND noninteractive
ENV BUILD_DIR ${GOPATH}/src/github.com/JustaPenguin/assetto-server-manager
ENV GO111MODULE on

RUN sed -i 's/deb.debian.org/archive.debian.org/g' /etc/apt/sources.list && \
    sed -i 's/security.debian.org/archive.debian.org/g' /etc/apt/sources.list && \
    sed -i '/stretch-updates/d' /etc/apt/sources.list

RUN apt-get update && apt-get install -y build-essential libssl-dev curl tofrodos dos2unix zip

ADD . ${BUILD_DIR}
WORKDIR ${BUILD_DIR}
RUN rm -rf cmd/server-manager/typescript/node_modules
RUN VERSION=${SM_VERSION} make deploy
RUN mv cmd/server-manager/build/linux/server-manager /usr/bin/

FROM ubuntu:18.04 AS run
MAINTAINER Callum Jones <cj@icj.me>

ENV DEBIAN_FRONTEND noninteractive

ENV SERVER_USER assetto
ENV SERVER_MANAGER_DIR /home/${SERVER_USER}/server-manager/
ENV SERVER_INSTALL_DIR ${SERVER_MANAGER_DIR}/assetto
ENV LANG C.UTF-8

ENV STEAMCMD_URL="http://media.steampowered.com/installer/steamcmd_linux.tar.gz"
ENV STEAMROOT=/opt/steamcmd

# steamcmd
RUN curl -sL https://deb.nodesource.com/setup_11.x | bash -
RUN apt-get update && apt-get install -y build-essential libssl-dev curl lib32gcc1 lib32stdc++6 nodejs
RUN mkdir -p ${STEAMROOT}
WORKDIR ${STEAMROOT}
RUN curl -s ${STEAMCMD_URL} | tar -vxz
ENV PATH "${STEAMROOT}:${PATH}"

# update steam
RUN steamcmd.sh +login anonymous +quit; exit 0

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
