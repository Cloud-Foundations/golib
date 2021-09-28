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
type Config struct {
	// AuthURL specifies the authorisation endpoint of the IDP. This is not
	// needed for an OpenID-Connect IDP.
	AuthURL string `yaml:"auth_url" envconfig:"OIDC_AUTH_URL"`

	// ClientID specifies the ID of this client, registered with the IDP. This
	// is required.
	ClientID string `yaml:"client_id" envconfig:"OIDC_CLIENT_ID"`

	// ClientSecret specifies the client shared secret. This is required.
	ClientSecret string `yaml:"client_secret" envconfig:"OIDC_CLIENT_SECRET"`

	// MaxAuthCookieLifetime specifies the maximum lifetime of the
	// authentication cookie. This is optional (default 12 hours). The minimum
	// is 5 minutes and the maximum is 24 hours.
	MaxAuthCookieLifetime time.Duration `yaml:"max_auth_cookie_lifetime" envconfig:"OIDC_MAX_AUTH_COOKIE_LIFETIME"`

	// ProviderURL specifies the base URL of the IDP. This is required.
	ProviderURL string `yaml:"provider_url" envconfig:"OIDC_PROVIDER_URL"`

	// Scopes specifies the scopes to request. This is required.
	Scopes string `yaml:"scopes" envconfig:"OIDC_SCOPES"`

	// SharedSecretFilename specifies a file containing one or more secrets
	// which are used so that multiple instances of the web application can
	// trust each others authN cookies. If this is not specified then a
	// secret is generated when the application starts up, which will cause
	// existing authN cookies to be invalidated. If the file is empty then a
	// secret is generated and written to the file, so that existing authN
	// cookies are not invalidated upon restart.
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
