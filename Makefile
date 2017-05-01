all: push

BUILDTAGS=

RELEASE?=0.0.1
PREFIX?=wehco/caddy-ingress-controller
GOOS?=linux

build: clean
	CGO_ENABLED=0 GOOS=${GOOS} go build -a -installsuffix cgo \
		-ldflags "-s -w -X ${PKG}/pkg/version.RELEASE=${RELEASE} -X ${PKG}/pkg/version.COMMIT=${COMMIT} -X ${PKG}/pkg/version.REPO=${REPO_INFO}" \
		-o rootfs/caddy-ingress-controller ${PKG}/pkg/cmd/controller

# Build the project before the container
container: build
	docker build --pull -t $(PREFIX):$(RELEASE) rootfs

push: container
	docker push $(PREFIX):$(RELEASE)

test:
	echo "test"
clean:
	echo "clean"
