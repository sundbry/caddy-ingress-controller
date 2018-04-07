#
# Builder
#
FROM abiosoft/caddy:builder as builder

ARG version="0.10.12"
ARG plugins="git"

# process wrapper
RUN go get -v github.com/abiosoft/parent

RUN VERSION=${version} PLUGINS=${plugins} /bin/sh /usr/bin/builder.sh

# Credits to https://github.com/abiosoft/caddy-docker
FROM alpine:3.7
MAINTAINER Kevin Minehart <kminehart@wehco.com>

ARG plugins=http.jwt,http.proxyprotocol,http.realip

RUN apk add --no-cache openssh-client git tar curl

COPY --from=builder /install/caddy /usr/bin/caddy

EXPOSE 80 443 12015

# This is where TLS certificates from acme live.
VOLUME /root/.caddy

# This will copy the generated config files to the filesystem.
COPY . /

CMD ["/caddy-ingress-controller"]
