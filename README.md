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

Also, certificate persistent storage needs implemented. There are some vestiges
left from an attempt to mount a secret as a RW filesystem, but my understanding
of k8s is lacking in that regard.

## Disclaimer

Much of the code in this controller was copied from the nginx ingress
controller, which is noted at the top of the copied files.
