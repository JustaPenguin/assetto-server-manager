#!/bin/bash

SERVER_INSTALL_DIR=$HOME/server-manager/assetto
SETUP_FILE=${SERVER_INSTALL_DIR}/.cars-setup

if [[ ! -f ${SETUP_FILE} ]]; then
    ./content-structure.sh ${SERVER_INSTALL_DIR}
    touch ${SETUP_FILE}
fi

server-manager