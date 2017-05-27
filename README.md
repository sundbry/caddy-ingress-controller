# Caddy Ingress Controller

This is an Ingress Controller, which implements the `ingress.Controller`
interface provided by [k8s.io/ingress/core/pkg/ingress](https://github.com/kubernetes/ingress/tree/master/core/pkg/ingress).

The implementation is fairly incomplete, in that it does not support any of the
ingress annotations (yet!) but Caddy's straightforward configuration process
should make them reasonably easy to implement for someone familiar with k8s
networking.

Additionally, I have only done some limited testing of Caddy's automatic HTTPS
features which utilize LetsEncrypt to automatically fetch and implement TLS
certificates for hosts specified by Ingresses. I have managed to get it to work
a few times, but not reliably enough to be able to say that this controller
provides that feature at this time.

## Deployment

`go get` this repo, make sure you're signed into docker hub, then run

```
PREFIX={my-docker-username}/caddy-controller make push
```

A sample deployment.yaml has been included. Just replace the image name &
email address with your own.

## Configuration

**LetsEncrypt Email Address** - Passed into the container via the `ACME_EMAIL`
environment variable - currently set in the sample deployment.yaml.

**LetsEncrypt Server** - Currently set to Staging, which will provide test
certs if successful. To change to production, comment out the `-ca` arg
in `pkg/cmd/controller/caddy.go`. **WARNING** this controller is not yet ready
for production use and changing this will almost certainly cause you to hit your
LetsEncrypt rate limits, which will prevent you from getting new certs for your
domain for a whole week!

**Caddy Plugins** - There's a comma-delimited list of plugins at
`rootfs/Dockerfile`, just update that list before running `make push`

## Development Contributions

Kubernetes has already taken care of the hard part - parsing out the Ingress
definitions, organizing them, and automatically pinging the controller with
updates when the definitions change. That means that further development of
this controller mostly relies on understanding the [`ingress.Configuration`](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/types.go#L133)
struct received by the controller and mapping that out into a
[Caddyfile template.](https://github.com/wehco/caddy-ingress-controller/blob/master/rootfs/etc/Caddyfile.tmpl)

The controller currently logs the updated config and Caddyfile template output
to stdout for relatively easy debugging. The `-log stdout` argument is also
being passed to the Caddy process so that its process logs will be printed
as well.

## Roadmap

### Embedded Caddy

This controller was built primarily by following the nginx controller's
example, but the differences between the two servers means that's not terribly
ideal.

The current model has a controller process controlling a separate Caddy
process, which is how the nginx controller works. One difference in
implementation, however, is that Caddy doesn't need to restart the process
entirely in order to load a new configuration - a live reload of the config
is instead triggered by sending a USR1 signal to the Caddy process.

I think the model can be further improved by eliminating the secondary process
altogether, embedding Caddy into the controller itself:
https://github.com/mholt/caddy/wiki/Embedding-Caddy-in-your-Go-program

### Annotations

As defined by https://github.com/kubernetes/ingress/tree/master/core/pkg/ingress/annotations

| Supported | Name                                                                                                                                                   |
|-----------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
| ✘         | [ingress.kubernetes.io/auth-type](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/auth/main.go)                         |
| ✘         | [ingress.kubernetes.io/auth-secret](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/auth/main.go)                       |
| ✘         | [ingress.kubernetes.io/auth-realm](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/auth/main.go)                        |
| ✘         | [ingress.kubernetes.io/auth-url](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/authreq/main.go)                       |
| ✘         | [ingress.kubernetes.io/auth-signin](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/authreq/main.go)                    |
| ✘         | [ingress.kubernetes.io/auth-method](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/authreq/main.go)                    |
| ✘         | [ingress.kubernetes.io/auth-send-body](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/authreq/main.go)                 |
| ✘         | [ingress.kubernetes.io/auth-response-headers](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/authreq/main.go)          |
| ✘         | [ingress.kubernetes.io/auth-tls-secret](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/authtls/main.go)                |
| ✘         | [ingress.kubernetes.io/auth-tls-verify-depth](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/authtls/main.go)          |
| ✘         | [ingress.kubernetes.io/enable-cors](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/cors/main.go)                       |
| ✘         | [ingress.kubernetes.io/upstream-max-fails](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/healthcheck/main.go)         |
| ✘         | [ingress.kubernetes.io/upstream-fail-timeout](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/healthcheck/main.go)      |
| ✘         | [ingress.kubernetes.io/whitelist-source-range](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/ipwhitelist/main.go)     |
| ✘         | [ingress.kubernetes.io/use-port-in-redirects](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/portinredirect/main.go)   |
| ✘         | [ingress.kubernetes.io/proxy-body-size](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/proxy/main.go)                  |
| ✘         | [ingress.kubernetes.io/proxy-connect-timeout](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/proxy/main.go)            |
| ✘         | [ingress.kubernetes.io/proxy-send-timeout](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/proxy/main.go)               |
| ✘         | [ingress.kubernetes.io/proxy-read-timeout](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/proxy/main.go)               |
| ✘         | [ingress.kubernetes.io/proxy-buffer-size](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/proxy/main.go)                |
| ✘         | [ingress.kubernetes.io/proxy-cookie-path](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/proxy/main.go)                |
| ✘         | [ingress.kubernetes.io/proxy-cookie-domain](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/proxy/main.go)              |
| ✘         | [ingress.kubernetes.io/limit-connections](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/ratelimit/main.go)            |
| ✘         | [ingress.kubernetes.io/limit-rps](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/ratelimit/main.go)                    |
| ✘         | [ingress.kubernetes.io/rewrite-target](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/rewrite/main.go)                 |
| ✘         | [ingress.kubernetes.io/add-base-url](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/rewrite/main.go)                   |
| ✘         | [ingress.kubernetes.io/ssl-redirect](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/rewrite/main.go)                   |
| ✘         | [ingress.kubernetes.io/force-ssl-redirect](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/rewrite/main.go)             |
| ✘         | [ingress.kubernetes.io/app-root](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/rewrite/main.go)                       |
| ✘         | [ingress.kubernetes.io/secure-backends](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/secureupstream/main.go)         |
| ✘         | [ingress.kubernetes.io/secure-verify-ca-secret](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/secureupstream/main.go) |
| ✘         | [ingress.kubernetes.io/affinity](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/sessionaffinity/main.go)               |
| ✘         | [ingress.kubernetes.io/session-cookie-name](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/sessionaffinity/main.go)    |
| ✘         | [ingress.kubernetes.io/session-cookie-hash](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/sessionaffinity/main.go)    |
| ✘         | [ingress.kubernetes.io/configuration-snippet](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/snippet/main.go)          |
| ✘         | [ingress.kubernetes.io/ssl-passthrough](https://github.com/kubernetes/ingress/blob/master/core/pkg/ingress/annotations/sslpassthrough/main.go)         |

## Disclaimer

Much of the code in this controller was copied from the nginx ingress
controller, which is noted at the top of the copied files.
