FROM alpine:latest

MAINTAINER Callum Jones <cj@icj.me>

# user setup
ENV SERVER_USER assetto
ENV SERVER_MANAGER_DIR /home/${SERVER_USER}/server-manager/
ENV SERVER_INSTALL_DIR ${SERVER_MANAGER_DIR}/assetto
RUN adduser -D -s /bin/bash ${SERVER_USER}

# dependencies
RUN apk update && apk add ca-certificates

ADD cmd/server-manager/build/linux/server-manager /usr/bin/server-manager

# install
RUN mkdir -p ${SERVER_MANAGER_DIR}
RUN mkdir ${SERVER_INSTALL_DIR}

RUN chown -R ${SERVER_USER}:${SERVER_USER} ${SERVER_MANAGER_DIR}
RUN chown -R ${SERVER_USER}:${SERVER_USER} ${SERVER_INSTALL_DIR}

USER ${SERVER_USER}
WORKDIR ${SERVER_MANAGER_DIR}

# recommend volume mounting the entire assetto corsa directory
VOLUME ["${SERVER_INSTALL_DIR}"]
EXPOSE 8772
EXPOSE 9600
EXPOSE 8081

ENTRYPOINT ["/usr/bin/server-manager"]
