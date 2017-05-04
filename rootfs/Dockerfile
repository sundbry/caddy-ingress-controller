# Credits to https://github.com/abiosoft/caddy-docker
FROM alpine:3.5
MAINTAINER Kevin Minehart <kminehart@wehco.com>

ARG plugins=http.jwt,http.proxyprotocol

RUN curl --silent --show-error --fail --location \
      --header "Accept: application/tar+gzip, application/x-gzip, application/octet-stream" -o - \
      "https://caddyserver.com/download/linux/amd64?plugins=${plugins}" \
    | tar --no-same-owner -C /usr/bin/ -xz caddy \
 && chmod 0755 /usr/bin/caddy \
 && /usr/bin/caddy -version

EXPOSE 80 443 2015

# This is where TLS certificates from acme live.
VOLUME /root/.caddy

# This will copy the generated config files to the filesystem.
COPY . /

CMD ["/caddy-ingress-controller"]
