# set the version if it's not already set
VERSION?=$(shell date +'%y.%m.%d-%H-%M')

export GO111MODULE=on

clean:
	rm -rf server-manager server-manager.exe
	rm -rf static/manager.js
	rm -rf build/
	rm -rf rsrc.syso
	rm -rf views/static_embed.go
	rm -rf static/static_embed.go

run:
	node_modules/.bin/babel javascript/manager.js --watch --out-file static/manager.js  --source-maps &
	go build -ldflags "-s -w -X github.com/cj123/assetto-server-manager.BuildTime=${VERSION}"
	FILESYSTEM_HTML=true DEBUG=true ./server-manager

assets:
	npm install
	node_modules/.bin/babel javascript/manager.js -o static/manager.js

deploy: clean assets
	# asset embed
	go get -u github.com/mjibson/esc
	go generate ./...

	# directory structure
	mkdir -p build/linux build/windows
	cp config.example.yml build/linux/config.yml
	cp config.example.yml build/windows/config.yml

	# windows icon
	go get -u github.com/akavel/rsrc
	rsrc -ico ../../misc/windows/icon.ico -arch amd64 -o rsrc.syso

	# build
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X github.com/cj123/assetto-server-manager.BuildTime=${VERSION}" -o build/linux/server-manager
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X github.com/cj123/assetto-server-manager.BuildTime=${VERSION}" -o build/windows/server-manager.exe

	cp ../../README.md build/README.txt
	cp ../../INSTALL.txt build/INSTALL.txt
	cp ../../LICENSE build/LICENSE.txt
	cp ../../CHANGELOG.txt build
	unix2dos build/README.txt build/LICENSE.txt build/CHANGELOG.txt build/windows/config.yml build/INSTALL.txt

	# archive
	cd build && zip -r ../server-manager_${VERSION}.zip *