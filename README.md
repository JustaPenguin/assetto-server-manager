Assetto Server Manager
======================

[![Build Status](https://travis-ci.org/cj123/assetto-server-manager.svg?branch=master)](https://travis-ci.org/cj123/assetto-server-manager)

A web interface to manage an Assetto Corsa Server.

## Features

* Quick Race Mode
* Custom Race Mode with saved presets
* Live Timings for current sessions
* Results pages for all previous sessions
* Content Management - Upload tracks, weather and cars
* Championship mode - configure multiple race events and keep track of driver and team points
* Server Logs / Options Editing
* Accounts system with different permissions levels
* Linux and Windows Support!

## Installation

### Docker

A docker image is available under the name `seejy/assetto-server-manager`. We recommend using docker-compose
to set up a docker environment for the server manager. This docker image has steamcmd pre-installed.

See [Manual](#Manual) to set up server manager without Docker.

**Note**: if you are using a directory volume for the server install (as is shown below), be sure to make 
the directory before running `docker-compose up` - otherwise its permissions may be incorrect. 

You will need a [config.yml](https://github.com/cj123/assetto-server-manager/blob/master/cmd/server-manager/config.example.yml) file to mount into the docker container.

An example docker-compose.yml looks like this:

```yaml
version: "3"

services:
  server-manager:
    image: seejy/assetto-server-manager:latest
    ports:
    # the port that the server manager runs on
    - 8772:8772
    # the port that the assetto server runs on (may vary depending on your configuration inside server manager)
    - 9600:9600
    # the port that the assetto server HTTP API runs on.
    - 8081:8081
    # you may also wish to bind your configured UDP plugin ports here. 
    volumes: 
    # volume mount the entire server install so that 
    # content etc persists across restarts
    - ./server-install:/home/assetto/server-manager/assetto
    
    # volume mount the config
    - ./config.yml:/home/assetto/server-manager/config.yml
```

### Manual

1. Download the latest release from the [releases page](https://github.com/cj123/assetto-server-manager/releases)
2. Extract the release
3. Edit the config.yml to suit your preferences
4. Either:
   - Copy the server folder from your Assetto Corsa install into the directory you configured in config.yml, or
   - Make sure that you have [steamcmd](https://developer.valvesoftware.com/wiki/SteamCMD) installed and in your $PATH 
     and have configured the steam username and password in the config.yml file.
5. Start the server using `./server-manager` (on Linux) or by running `server-manager.exe` (on Windows)


### Post Installation

We recommend uploading your entire Assetto Corsa `content/tracks` folder to get the full features of Server Manager. 
This includes things like track images, all the correct layouts and any mod tracks you may have installed.

Also, we recommend installing Sol locally and uploading your Sol weather files to Server Manager as well so you can try out Day/Night cycles and cool weather!

## Credits & Thanks

Assetto Corsa Server Manager would not have been possible without the following people:

* Henry Spencer - [Twitter](https://twitter.com/HWSpencer) / [GitHub](https://github.com/Hecrer)
* Callum Jones - [Twitter](https://twitter.com/icj_) / [GitHub](https://github.com/cj123)
* Joseph Elton
* The Pizzabab Championship
* [ACServerManager](https://github.com/Pringlez/ACServerManager) and its authors, for 
inspiration and reference on understanding the AC configuration files

## Screenshots

Check out the [screenshots folder](https://github.com/cj123/assetto-server-manager/tree/master/misc/screenshots)!