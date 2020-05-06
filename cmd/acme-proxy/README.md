# acme-proxy
ACME Proxy.

The *acme-proxy* will cache and/or forward
[ACME](https://en.wikipedia.org/wiki/Automated_Certificate_Management_Environment)
http-01 challenge-response requests. It is typically used to allow certificate
managers for Web servers which are not publicly accessible to request X.509
certificates from a public Certificate Authority such as
[Let's Encrypt](https://letsencrypt.org/).

The certificate manager may be integrated in the Web server or may be an
external server such as [certmanager](../certmanager/README.md). The certificate
manager is the component which issues ACME requests and must respond to http-01
challenge-response requests.

The *acme-proxy* expects to be run in a
[split-horizon DNS](https://en.wikipedia.org/wiki/Split-horizon_DNS)
environment. Every FQDN for which X.509 certificates will be requested must
resolve to the *acme-proxy* in the external (public Internet) DNS view and must
resolve to the Web server certificate manager in the internal DNS view which
*acme-proxy* sees.

## Caching mode
If the certificate manager is based on the
[certmanager](../../pkg/crypto/certmanager/) package then it will upload http-01
challenge responses to *acme-proxy* which will in turn respond with these
cached responses. The *acme-proxy* expands the list of IP addresses for the
request (the Web server host) and checks for a match with the IP address of the
certificate manager which uploaded the response. This mode of operation is
preferred as it does not require *acme-proxy* to connect to the back-end
servers, thus supporting the highest level of security.

## Forwarding mode
If a certificate manager does not support the caching protocol, then
*acme-proxy* will automatically fall back to simple forwarding of the
challenge-response requests.

It is not necessary to configure *acme-proxy* to direct where to forward
http-01 challenge-response requests, instead, *acme-proxy* uses the internal DNS
iew to determine where to forward requests to.

Only http-01 challenge-response requests are forwarded by *acme-proxy*. No other
requests are forwarded, keeping internal Web servers safe from hostile traffic.
In addition, the requests are forwarded by issuing new HTTP requests, rather
than forwarding raw TCP traffic.

## Status page
The *acme-proxy* provides a web interface on port `6941` which shows a status
page, links to built-in dashboards and access to performance metrics and logs.
If *acme-proxy* is running on host `myhost` then the URL of the main
status page is `http://myhost:6941/`.

## Configuration
Configuration is performed using command-line flags. There are command-line
flags which may change the behaviour of *acme-proxy* but many have defaults
which should be adequate for most deployments. Built-in help is available with
the command:

```
acme-proxy -h
```

The `/etc/acme-proxy/flags.default` and `/etc/acme-proxy/flags.extra` files are
read at startup (in that order), overriding built-in defaults. Options given on
the command-line are processed last (and take precedence).

## ACME port number
The ACME protocol requires that http-01 challenge-response requests are sent to
the standard HTTP port 80. The *acme-proxy* will listen on this port by default.
If your firewall/router redirects incoming connections to a different port (i.e.
8080), use the following option to change the listening port number:

```
-acmePortNum=8080
```

## Certificate manager port number
If you are running your certificate manager for your Web server on a different
port than 80, you may configure *acme-proxy* to forward the requests to a
different port if a HTTP 404 (Not Found) error is received by *acme-proxy* when
forwarding to port 80. For example, if your certificate manager is running on
port 8080, use the following option

```
-fallbackPortNum=8080
```

This configuration allows you to co-host a certificate manager and Web server on
the same system, allowing the Web server to continue processing HTTP requests in
addition to HTTPS requests.
