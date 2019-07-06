
# enable go modules
GO111MODULE=on

all: clean test assets build

clean:
	$(MAKE) -C cmd/server-manager clean

test:
	mkdir -p cmd/server-manager/assetto/cfg
	mkdir -p cmd/server-manager/assetto/results
	cp -R fixtures/results/*.json cmd/server-manager/assetto/results
	go test

assets:
	$(MAKE) -C cmd/server-manager assets

asset-embed:
	$(MAKE) -C cmd/server-manager asset-embed

build:
	$(MAKE) -C cmd/server-manager build

deploy: clean test
	$(MAKE) -C cmd/server-manager deploy

run:
	$(MAKE) -C cmd/server-manager run