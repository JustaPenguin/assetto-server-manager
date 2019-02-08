Assetto Server Manager
======================

A web interface to manage an Assetto Corsa Server.

## Features

* Quick Race Mode
* Custom Race Mode with saved presets
* Results pages for all previous sessions (live timings coming soon!)
* Content Management - Upload tracks, weather and cars
* Championship mode ([watch this space!](https://github.com/cj123/assetto-server-manager/issues/4))
* Server Logs / Options Editing

![Server Manager](https://static-dl.justapengu.in/server_manager.png)


## Installation

### Docker

A docker image is available under the name `seejy/assetto-server-manager`. We recommend using docker-compose
to set up a docker environment for the server manager. This docker image has steamcmd pre-installed.

See [Manual](#Manual) to set up server manager without Docker.

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
    volumes: 
    # volume mount the entire server install so that 
    # content etc persists across restarts
    - ./server-install:/home/assetto/server-manager/assetto
    environment:
    # see cmd/server-manager/env.example for a full list of these options
    
    # steam username and password. we recommend creating a separate account with
    # steamguard disabled to use this application.
    # server-manager uses this information ONLY to install the assetto corsa server.
    - STEAM_USERNAME=steamuser
    - STEAM_PASSWORD=hunter2

    # a key to be used to encrypt session values. we recommend randomly generating this!
    - SESSION_KEY=super_secret_session_value
```

### Manual

1. Make sure that you have [steamcmd](https://developer.valvesoftware.com/wiki/SteamCMD) installed and in your $PATH
2. Download the latest release from the [releases page](https://github.com/cj123/assetto-server-manager/releases)
3. Extract the release
4. Copy env.example to .env in the same directory, and edit its values
5. Start the server using `./run.sh`

## Credits & Thanks

Assetto Corsa Server Manager would not have been possible without the following people:

* Henry Spencer - [Twitter](https://twitter.com/HWSpencer) / [GitHub](https://github.com/Hecrer)
* Callum Jones - [Twitter](https://twitter.com/icj_) / [GitHub](https://github.com/cj123)
* Joseph Elton
* The Pizzabab Championship
* [ACServerManager](https://github.com/Pringlez/ACServerManager) and its authors, for 
inspiration and reference on understanding the AC configuration files