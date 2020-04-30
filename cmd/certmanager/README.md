# certmanager
X.509 Web Certificate Manager.

The *certmanager* issues requests and periodic renewals for X.509 certificates
suitable for Web (HTTPS) services. It is a simple wrapper around the
[certmanager package](../../pkg/crypto/certmanager). This package supports
using remote locks and secrets managers so that a cluster of Web servers may
safely request and share a public certificate, without exhausting certificate
request quotas.

It may be used for debugging code and configuration as well as run as a standard
system daemon to provide certificates for a Web server (i.e. Apache). In the
latter case it provides some of the features of
[certbot](https://certbot.eff.org/) however it is simpler to configure and has
the above mentioned capabilities to safely request certificates for a cluster of
Web servers.

## Status page
The *certmanager* provides a web interface on port `6940` which shows a status
page, links to built-in dashboards and access to performance metrics and logs.
If *certmanager* is running on host `myhost` then the URL of the main
status page is `http://myhost:6940/`.

## Configuration
Configuration is performed using command-line flags. There are many command-line
flags which may change the behaviour of *certmanager* but many have defaults
which should be adequate for most deployments. Built-in help is available with
the command:

```
certmanager -h
```

The `/etc/certmanager/flags.default` and `/etc/certmanager/flags.extra` are read
at startup (in that order), overriding built-in defaults. Options given on the
command-line are processed last (and take precedence).

## Debugging (command-line) mode
In this mode you may prefer to receive logs on the standard error and not write
to a logfile. The following options are recommended:

```
-alsoLogToStderr=true -logDir=
```

Note that even in debugging mode, *certmanager* will run until interrupted,
requesting new certificates periodically (about every 60 days).

## Daemon (server) mode
By default *certmanager* will request _testing_ certificates which are not
trusted. This default is intended to prevent the accidental exhaustion of
certificate request quota (5 per FQDN per week with
[Let's Encrypt](https://letsencrypt.org/)). Once you are confident of your
configuration, use the following option:

```
-production=true
```

## Restarting a service
If you are running *certmanager* to provide certificates for a Web server such
as Apache, use the following option:

```
-notifierCommand='service apache reload'
```

## Redirecting HTTP to HTTPS
If you wish to redirect HTTP requests to the HTTPS Web server, use the following
option:

```
-redirect=true
```
