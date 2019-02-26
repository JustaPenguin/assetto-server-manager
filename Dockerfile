FROM golang:1.11.5

MAINTAINER Callum Jones <cj@icj.me>

ENV STEAMCMD_URL="http://media.steampowered.com/installer/steamcmd_linux.tar.gz"
ENV STEAMROOT=/opt/steamcmd
ENV DEBIAN_FRONTEND noninteractive
ENV SERVER_USER assetto
ENV BUILD_DIR ${GOPATH}/src/github.com/cj123/assetto-server-manager
ENV SERVER_MANAGER_DIR /home/${SERVER_USER}/server-manager/
ENV SERVER_INSTALL_DIR ${SERVER_MANAGER_DIR}/assetto
ENV GO111MODULE on

# steamcmd
RUN apt-get update && apt-get install -y curl lib32gcc1 lib32stdc++6
RUN mkdir -p ${STEAMROOT}
WORKDIR ${STEAMROOT}
RUN curl -s ${STEAMCMD_URL} | tar -vxz
ENV PATH "${STEAMROOT}:${PATH}"

# update steam
RUN steamcmd.sh +login anonymous +quit; exit 0

# build
ADD . ${BUILD_DIR}
WORKDIR ${BUILD_DIR}/cmd/server-manager
RUN go get github.com/mjibson/esc
RUN go generate ./...
RUN export BUILD_TIME=`date +'%s'`; go build -ldflags "-s -w -X github.com/cj123/assetto-server-manager.BuildTime=$BUILD_TIME"
RUN mv server-manager /usr/bin/

RUN useradd -ms /bin/bash ${SERVER_USER}

# install
RUN mkdir -p ${SERVER_MANAGER_DIR}
RUN mkdir ${SERVER_INSTALL_DIR}

RUN chown -R ${SERVER_USER}:${SERVER_USER} ${SERVER_MANAGER_DIR}
RUN chown -R ${SERVER_USER}:${SERVER_USER} ${SERVER_INSTALL_DIR}

# cleanup
RUN rm -rf ${BUILD_DIR}

USER ${SERVER_USER}
WORKDIR ${SERVER_MANAGER_DIR}

# recommend volume mounting the entire assetto corsa directory
VOLUME ["${SERVER_INSTALL_DIR}"]
EXPOSE 8772
EXPOSE 9600
EXPOSE 8081

ENTRYPOINT ["server-manager"]
