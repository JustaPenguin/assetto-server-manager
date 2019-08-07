VERSION?=unstable

# enable go modules
export GO111MODULE=on

all: clean test assets build

clean:
	rm -rf changelog_embed.go
	$(MAKE) -C cmd/server-manager clean

test:
	mkdir -p cmd/server-manager/assetto/cfg
	mkdir -p cmd/server-manager/assetto/results
	cp -R fixtures/results/*.json cmd/server-manager/assetto/results
	go test -mod vendor

generate:
	go get -u github.com/mjibson/esc
	go generate -mod vendor .

assets:
	$(MAKE) -C cmd/server-manager assets

asset-embed:
	$(MAKE) -C cmd/server-manager asset-embed

build:
	$(MAKE) -C cmd/server-manager build

deploy: clean generate test
	$(MAKE) -C cmd/server-manager deploy

run:
	$(MAKE) -C cmd/server-manager run

docker:
	docker build --build-arg SM_VERSION=${VERSION} -t seejy/assetto-server-manager:${VERSION} .
	docker push seejy/assetto-server-manager:${VERSION}