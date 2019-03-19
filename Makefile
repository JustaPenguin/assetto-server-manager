
# enable go modules
GO111MODULE=on

all: clean test assets build

clean:
	$(MAKE) -C cmd/server-manager clean

test:
	mkdir -p cmd/server-manager/assetto/{cfg,results}
	cp -R fixtures/results/*.json cmd/server-manager/assetto/results
	go test

assets:
	$(MAKE) -C cmd/server-manager assets

build:
	$(MAKE) -C cmd/server-manager build

deploy: clean test
	$(MAKE) -C cmd/server-manager deploy
