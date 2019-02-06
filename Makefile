# This file is a derivative of https://github.com/kubernetes/ingress/blob/master/controllers/nginx/Makefile
# Licensed under the Apache License 2.0 http://www.apache.org/licenses/LICENSE-2.0

all: push

BUILDTAGS=

# Use the 0.0 tag for testing, it shouldn't clobber any release builds
RELEASE?=0.0.1
PREFIX?=zikes/caddy-ingress
GOOS?=linux

REPO_INFO=$(shell git config --get remote.origin.url)

ifndef COMMIT
  COMMIT := git-$(shell git rev-parse --short HEAD)
endif

PKG=k8s.io/ingress/controllers/caddy

caddy-ingress-controller:
	CGO_ENABLED=0 GOOS=${GOOS} go build -a -installsuffix cgo \
		-ldflags "-s -w -X ${PKG}/pkg/version.RELEASE=${RELEASE} -X ${PKG}/pkg/version.COMMIT=${COMMIT} -X ${PKG}/pkg/version.REPO=${REPO_INFO}" \
		-o caddy-ingress-controller ${PKG}/pkg/cmd/controller

container:
	docker build --pull -t $(PREFIX):$(RELEASE) \
		--build-arg RELEASE=$(RELEASE) \
		--build-arg PKG=$(PKG) \
		--build-arg COMMIT=$(COMMIT) \
		.

debug-container: caddy-ingress-controller
	docker build --pull -t $(PREFIX):$(RELEASE) \
		-f Dockerfile.debug \
		.

push: 
	docker push $(PREFIX):$(RELEASE)

fmt:
	@echo "+ $@"
	@go list -f '{{if len .TestGoFiles}}"gofmt -s -l {{.Dir}}"{{end}}' $(shell go list ${PKG}/... | grep -v vendor) | xargs -L 1 sh -c

lint:
	@echo "+ $@"
	@go list -f '{{if len .TestGoFiles}}"golint {{.Dir}}/..."{{end}}' $(shell go list ${PKG}/... | grep -v vendor) | xargs -L 1 sh -c

test: fmt lint vet
	@echo "+ $@"
	@go test -v -race -tags "$(BUILDTAGS) cgo" $(shell go list ${PKG}/... | grep -v vendor)

cover:
	@echo "+ $@"
	@go list -f '{{if len .TestGoFiles}}"go test -coverprofile={{.Dir}}/.coverprofile {{.ImportPath}}"{{end}}' $(shell go list ${PKG}/... | grep -v vendor) | xargs -L 1 sh -c
	gover
	goveralls -coverprofile=gover.coverprofile -service travis-ci -repotoken ${COVERALLS_TOKEN}

vet:
	@echo "+ $@"
	@go vet $(shell go list ${PKG}/... | grep -v vendor)

clean:
	rm -f rootfs/caddy-ingress-controller
