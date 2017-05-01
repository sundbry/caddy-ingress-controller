# Caddy Ingress Controller

This is a Caddy ingress controller for Kubernetes.

Caddy is a simple http server written in Go. [Read more here.](https://github.com/mholt/caddy)

## Using the ingress controller

When creating an ingress, specify the annotation:

```
kubernetes.io/ingress.class: "caddy"
```

Example:

```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: caddy-test
  annotations:
    kubernetes.io/ingress.class: "caddy"
spec:
  rules:
  - host: myhost.example.com
    http:
      paths:
      - path: /
        backend:
          serviceName: my-service
          servicePort: 9090
```

## Configuration

You can also add the following annotations for various settings.

| Name | type |
|------|------|
| [ingress.kubernetes.io/jwt](#jwt) | true or false |
| [ingress.kubernetes.io/jwt-path](#jwt) | string |
| [ingress.kubernetes.io/jwt-redirect](#jwt) | string |
| [ingress.kubernetes.io/jwt-allow](#jwt) | string |
| [ingress.kubernetes.io/jwt-deny](#jwt) | string |

# jwt

A JSON Web Token (JWT) is a secure authentication token that stores data.

*WARNING* When jwt is enabled, the private key must be deployed with the ingress controller via. setting the environment variable `JWT_SECRET` (HMAC) or `JWT_PUBLIC_KEY` (RSA).

Read more about the jwt plugin [here](https://github.com/BTBurke/caddy-jwt)

### ingress.kubernetes.io/jwt

To enable jwt, set

```
ingress.kubernetes.io/jwt: "true"
```

### ingress.kubernetes.io/jwt-path

The `jwt-path` is the path on the server that requires a valid JWT.

By default, setting `ingress.kubernetes.io/jwt` to true will also set 

```
ingress.kubernetes.io/jwt-path: "/"
```
