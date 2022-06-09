/*
Package oidc implements an OpenID-Connect/OAuth2 client (Service Provider)
wrapper. A web application may use this to enforce authentication and
authorisation using a specified OpenID-Connect/OAuth2 IDentity Provider (IDP).
*/
package oidc

import (
	"net/http"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/auth/authinfo"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

// Config specifies the client OpenID-Connect/OAuth2 configuration.
// Shared secrets are used so that multiple instances of the web application can
// trust each others authN cookies. If the instance cannot share these secrets
// there may be increased latency as the browser is bounced between instances,
// needing to re-request credentials from the IDentity Provider. If no method
// for fetching the shared secrets is specified then a secret is generated when
// the application starts up, which will cause existing authN cookies to be
// invalidated.
type Config struct {
	// AuthURL specifies the authorisation endpoint of the IDP. This is not
	// needed for an OpenID-Connect IDP.
	AuthURL string `yaml:"auth_url" envconfig:"OIDC_AUTH_URL"`

	// AwsSecretId specifies the AWS secret containing the shared secrets. If
	// the secret object is empty then a secret is generated and saved.
	// Optional.
	AwsSecretId string `yaml:"aws_secret_id" envconfig:"OIDC_AWS_SECRET_ID"`

	// ClientID specifies the ID of this client, registered with the IDP. This
	// is required.
	ClientID string `yaml:"client_id" envconfig:"OIDC_CLIENT_ID"`

	// ClientSecret specifies the client shared secret. This is required.
	ClientSecret string `yaml:"client_secret" envconfig:"OIDC_CLIENT_SECRET"`

	// LoginCookieLifetime specifies the lifetime of the login cookie.
	// This is optional (default is no explicit login request is required).
	// The minimum is the greater of (MaxAuthCookieLifetime, 1 hour) and the
	// maximum is 400 days.
	LoginCookieLifetime time.Duration `yaml:"login_cookie_lifetime" envconfig:"OIDC_LOGIN_COOKIE_LIFETIME"`

	// MaxAuthCookieLifetime specifies the maximum lifetime of the
	// authentication cookie. This is optional (default 12 hours). The minimum
	// is 5 minutes and the maximum is 24 hours.
	MaxAuthCookieLifetime time.Duration `yaml:"max_auth_cookie_lifetime" envconfig:"OIDC_MAX_AUTH_COOKIE_LIFETIME"`

	// ProviderURL specifies the base URL of the IDP. This is required.
	ProviderURL string `yaml:"provider_url" envconfig:"OIDC_PROVIDER_URL"`

	// Scopes specifies the scopes to request. This is required.
	Scopes string `yaml:"scopes" envconfig:"OIDC_SCOPES"`

	// SharedSecretFilename specifies a file containing the shared secrets. If
	// the file is missing then a secret is generated and written to the file.
	// Optional.
	SharedSecretFilename string `yaml:"shared_secret_filename" envconfig:"OIDC_SHARED_SECRET_FILENAME"`

	// TokenURL specifies the token endpoint of the IDP. This is not needed for
	// an OpenID-Connect IDP.
	TokenURL string `yaml:"token_url" envconfig:"OIDC_TOKEN_URL"`

	// UserinfoURL specifies the userinfo endpoint of the IDP. This is not
	// needed for an OpenID-Connect IDP.
	UserinfoURL string `yaml:"userinfo_url" envconfig:"OIDC_USERINFO_URL"`
}

// Params specifies runtime parameters.
type Params struct {
	// AddHeaders specifies whether to add authentication headers to requests.
	// This can be useful if the HTTP request is forwarded to another server
	// (such as when using this package in a reverse authenticating proxy).
	AddHeaders bool

	// Handler specifies the HTTP handler for the application. This is only
	// used when the user is authenticated.
	Handler http.Handler

	// Logger specifies the logger to use.
	Logger log.DebugLogger

	// LogoutHandler specifies an optional handler to call when the user logs
	// out by visiting the "/logout" path. If this is not specified, a simple
	// default page is shown, informing the user they are logged out and showing
	// a button allowing them to log in.
	LogoutHandler func(w http.ResponseWriter, req *http.Request)
}

// NewAuthNHandler creates a new HTTP handler which handles all incoming HTTP
// requests. It will ensure the user is authenticated before passing HTTP
// requests to the application handler.
func NewAuthNHandler(config Config, params Params) (http.Handler, error) {
	return newAuthNHandler(config, params)
}

// GetAuthInfoFromRequest will return authentication information for the user
// for the specified HTTP request. It will return nil for a request that did
// not come through a handler returned from NewAuthNHandler.
func GetAuthInfoFromRequest(req *http.Request) *authinfo.AuthInfo {
	return getAuthInfoFromRequest(req)
}
