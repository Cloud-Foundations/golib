package oidc

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/auth/authinfo"
	"github.com/Cloud-Foundations/golib/pkg/auth/authinfo/x509util"
	"github.com/Cloud-Foundations/golib/pkg/constants"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

const (
	cookieNamePrefix              = "authn_cookie"
	secondsBetweenCleanup         = 60
	cookieExpirationHours         = 3
	maxAgeSecondsRedirCookie      = 120
	redirCookieName               = "oauth2_redir"
	oauth2redirectPath            = "/oauth2/redirectendpoint"
	authNCookieExpirationDuration = 12 * time.Hour
)

type accessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

type authNHandler struct {
	authCookieName   string
	config           Config
	netClient        *http.Client
	params           Params
	sharedSecrets    []string
	mutex            sync.Mutex                // Protect everything below.
	cachedUserGroups map[string]expiringGroups // Key: username.
}

type authInfo struct {
	Username   string   `json:"username,omitempty"`
	Issuer     string   `json:"iss,omitempty"`
	Subject    string   `json:"sub,omitempty"`
	Audience   []string `json:"aud,omitempty"`
	Expiration int64    `json:"exp,omitempty"` // Seconds since the Epoch.
	NotBefore  int64    `json:"nbf,omitempty"`
}

type expiringGroups struct {
	expires time.Time
	groups  []string
}

type oauth2StateJWT struct {
	Issuer     string   `json:"iss,omitempty"`
	Subject    string   `json:"sub,omitempty"`
	Audience   []string `json:"aud,omitempty"`
	Expiration int64    `json:"exp,omitempty"`
	NotBefore  int64    `json:"nbf,omitempty"`
	IssuedAt   int64    `json:"iat,omitempty"`
	ReturnURL  string   `json:"return_url,omitempty"`
}

type openidConnectUserInfo struct {
	Subject           string   `json:"sub"`
	Name              string   `json:"name"`
	Login             string   `json:"login,omitempty"`
	Username          string   `json:"username,omitempty"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
	Email             string   `json:"email,omitempty"`
	Groups            []string `json:"groups,omitempty"`
}

type openidConnectProviderConfiguration struct {
	AuthURL     string `json:"authorization_endpoint"`
	TokenURL    string `json:"token_endpoint"`
	UserinfoURL string `json:"userinfo_endpoint"`
}

type usernameSetter interface {
	SetUsername(username string)
}

func getAuthInfoFromRequest(req *http.Request) *authinfo.AuthInfo {
	return authinfo.GetAuthInfoFromContext(req.Context())
}

func getUsernameFromUserinfo(userInfo openidConnectUserInfo) string {
	username := userInfo.Username
	if len(username) < 1 {
		username = userInfo.Login
	}
	if len(username) < 1 {
		username = userInfo.PreferredUsername
	}
	if len(username) < 1 {
		username = userInfo.Email
	}
	return username
}

func newAuthNHandler(config Config, params Params) (*authNHandler, error) {
	h := &authNHandler{
		authCookieName:   cookieNamePrefix,
		config:           config,
		netClient:        &http.Client{Timeout: time.Second * 15},
		params:           params,
		cachedUserGroups: make(map[string]expiringGroups),
	}
	if err := h.loadSharedSecrets(); err != nil {
		return nil, err
	}
	if err := h.populateEndpoints(); err != nil {
		return nil, err
	}
	return h, nil
}

// Generates a valid auth cookie that can be used by clients, should only be
// used by users of the lib in their test functions
func (h *authNHandler) GenValidAuthCookie(userInfo *openidConnectUserInfo) (
	*http.Cookie, error) {
	expires := time.Now().Add(time.Hour * cookieExpirationHours)
	return h.genValidAuthCookieExpiration(userInfo, expires, "localhost")
}

func (h *authNHandler) genValidAuthCookieExpiration(
	userInfo *openidConnectUserInfo, expires time.Time, issuer string) (
	*http.Cookie, error) {
	key := []byte(h.sharedSecrets[0])
	sig, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS256,
		Key:       key,
	},
		(&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return nil, fmt.Errorf("genValidAuthCookieExpiration: %s", err)
	}
	subject := "state:" + h.authCookieName
	authToken := authInfo{
		Issuer:     issuer,
		Subject:    subject,
		Audience:   []string{issuer},
		Username:   userInfo.Username,
		Expiration: expires.Unix(),
	}
	// TODO: add IssuedAt and NotBefore?
	authToken.NotBefore = time.Now().Unix()
	//stateToken.IssuedAt = stateToken.NotBefore
	cookieValue, err := jwt.Signed(sig).Claims(authToken).CompactSerialize()
	if err != nil {
		return nil, err
	}
	userCookie := http.Cookie{
		Name:     h.authCookieName,
		Value:    cookieValue,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
	}
	return &userCookie, nil
}

// Returns cached group information for the user. If there is no valid cached
// information, returns false.
func (h *authNHandler) getCachedUserGroups(username string) (bool, []string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	eg, ok := h.cachedUserGroups[username]
	if !ok {
		return false, nil
	}
	if time.Since(eg.expires) > 0 {
		delete(h.cachedUserGroups, username)
		return false, nil
	}
	return true, eg.groups
}

func (h *authNHandler) setCachedUserGroups(username string, groups []string,
	expires time.Time) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if len(groups) < 1 {
		delete(h.cachedUserGroups, username)
		return
	}
	h.cachedUserGroups[username] = expiringGroups{
		expires: expires,
		groups:  groups,
	}
}

func (h *authNHandler) setAndStoreAuthCookie(w http.ResponseWriter,
	r *http.Request, userInfo *openidConnectUserInfo, expires time.Time) error {
	userCookie, err := h.genValidAuthCookieExpiration(userInfo, expires, r.Host)
	if err != nil {
		return err
	}
	h.params.Logger.Debugf(2,
		"setAndStoreAuthCookie: %s host: %s query: %s name: %s\n",
		r.URL.Path, r.Host, r.URL.RawQuery, userCookie.Name)
	http.SetCookie(w, userCookie)
	h.setCachedUserGroups(userInfo.Username, userInfo.Groups, expires)
	return nil
}

func (h *authNHandler) getRedirURL(r *http.Request) string {
	return "https://" + r.Host + oauth2redirectPath
}

func (h *authNHandler) generateAuthCodeURL(state string,
	r *http.Request) string {
	h.params.Logger.Debugf(2, "generateAuthCodeURL: %s query: %s\n",
		r.URL.Path, r.URL.RawQuery)
	var buf bytes.Buffer
	buf.WriteString(h.config.AuthURL)
	redirectURL := h.getRedirURL(r)
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {h.config.ClientID},
		"scope":         {h.config.Scopes},
		"redirect_uri":  {redirectURL},
	}
	if state != "" {
		// TODO(light): Docs say never to omit state; don't allow empty.
		v.Set("state", state)
	}
	if strings.Contains(h.config.AuthURL, "?") {
		buf.WriteByte('&')
	} else {
		buf.WriteByte('?')
	}
	buf.WriteString(v.Encode())
	return buf.String()
}

func (h *authNHandler) generateSharedSecrets() error {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	h.sharedSecrets = []string{base64.StdEncoding.EncodeToString(buf)}
	h.params.Logger.Println("Generated shared secrets")
	return nil
}

func (h *authNHandler) generateValidStateString(r *http.Request) (
	string, error) {
	key := []byte(h.sharedSecrets[0])
	sig, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS256,
		Key:       key,
	}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		h.params.Logger.Printf("err=%s", err)
		return "", err
	}
	issuer := r.Host
	subject := "state:" + redirCookieName
	stateToken := oauth2StateJWT{
		Issuer:    issuer,
		Subject:   subject,
		Audience:  []string{issuer},
		ReturnURL: r.URL.String(),
	}
	h.params.Logger.Debugf(2,
		"generateValidStateString: issuer: %s subject: %s\n", issuer, subject)
	stateToken.NotBefore = time.Now().Unix()
	stateToken.IssuedAt = stateToken.NotBefore
	stateToken.Expiration = stateToken.IssuedAt + maxAgeSecondsRedirCookie
	return jwt.Signed(sig).Claims(stateToken).CompactSerialize()
}

// This is where the redirect to the oath2 provider is computed.
func (h *authNHandler) oauth2DoRedirectoToProviderHandler(w http.ResponseWriter,
	r *http.Request) {
	h.params.Logger.Debugf(2,
		"oauth2DoRedirectoToProviderHandler: %s query: %s\n",
		r.URL.Path, r.URL.RawQuery)
	stateString, err := h.generateValidStateString(r)
	if err != nil {
		h.params.Logger.Printf("err=%s", err)
		http.Error(w, "Internal Error ", http.StatusInternalServerError)
		return
	}
	redirectURL := h.generateAuthCodeURL(stateString, r)
	h.params.Logger.Debugf(2,
		"oauth2DoRedirectoToProviderHandler: redirecting to: %s\n", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// Next are the functions fo checking the callback
func (h *authNHandler) JWTClaims(t *jwt.JSONWebToken, dest ...interface{}) (
	err error) {
	for _, key := range h.sharedSecrets {
		binkey := []byte(key)
		err = t.Claims(binkey, dest...)
		if err == nil {
			return nil
		}
	}
	if err != nil {
		return err
	}
	err = errors.New("No valid key found")
	return err
}

func (h *authNHandler) getBytesFromSuccessfullPost(url string,
	data url.Values) ([]byte, error) {
	response, err := h.netClient.PostForm(url, data)
	if err != nil {
		h.params.Logger.Printf("err=%s", err)
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		h.params.Logger.Printf("err=%s", err)
		return nil, err
	}
	if response.StatusCode >= 300 {
		h.params.Logger.Printf(string(responseBody))
		return nil, errors.New("invalid status code")
	}
	return responseBody, nil
}

func (h *authNHandler) getVerifyReturnStateJWT(r *http.Request) (
	oauth2StateJWT, error) {
	inboundJWT := oauth2StateJWT{}
	serializedState := r.URL.Query().Get("state")
	if len(serializedState) < 1 {
		return inboundJWT, errors.New("null inbound state")
	}
	tok, err := jwt.ParseSigned(serializedState)
	if err != nil {
		return inboundJWT, err
	}
	if err := h.JWTClaims(tok, &inboundJWT); err != nil {
		h.params.Logger.Printf("error parsing claims err=%s", err)
		return inboundJWT, err
	}
	// At this point we know the signature is valid, but now we must
	// validate the contents of the JWT token
	issuer := r.Host
	subject := "state:" + redirCookieName
	if inboundJWT.Issuer != issuer || inboundJWT.Subject != subject ||
		inboundJWT.NotBefore > time.Now().Unix() ||
		inboundJWT.Expiration < time.Now().Unix() {
		return inboundJWT, errors.New("invalid JWT values")
	}
	h.params.Logger.Debugf(1,
		"getVerifyReturnStateJWT: inbound JWT expiration: %s\n",
		time.Unix(inboundJWT.Expiration, 0))
	return inboundJWT, nil
}

func (h *authNHandler) loadSharedSecrets() error {
	if h.config.SharedSecretFilename == "" {
		return h.generateSharedSecrets()
	}
	if file, err := os.Open(h.config.SharedSecretFilename); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := h.generateSharedSecrets(); err != nil {
			return err
		}
		h.params.Logger.Printf("Writing shared secrets to file: %s\n",
			h.config.SharedSecretFilename)
		return ioutil.WriteFile(h.config.SharedSecretFilename,
			[]byte(h.sharedSecrets[0]+"\n"), 0600)
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			h.sharedSecrets = append(h.sharedSecrets, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		h.params.Logger.Printf("Read shared secrets from file: %s\n",
			h.config.SharedSecretFilename)
		return nil
	}
}

func (h *authNHandler) oauth2RedirectPathHandler(w http.ResponseWriter,
	r *http.Request) {
	h.params.Logger.Debugf(2, "oauth2RedirectPathHandler: %s query: %s\n",
		r.URL.Path, r.URL.RawQuery)
	if r.Method != "GET" {
		h.params.Logger.Printf("Bad method on redirect, should only be GET")
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}
	authCode := r.URL.Query().Get("code")
	if len(authCode) < 1 {
		h.params.Logger.Println("null code")
		http.Error(w, "null code", http.StatusUnauthorized)
		return
	}
	inboundJWT, err := h.getVerifyReturnStateJWT(r)
	if err != nil {
		h.params.Logger.Printf("error processing state err=%s", err)
		http.Error(w, "null or bad inboundState", http.StatusUnauthorized)
		return
	}
	h.params.Logger.Debugf(1,
		"oauth2RedirectPathHandler: inboundJWT expires: %s\n",
		time.Unix(inboundJWT.Expiration, 0))
	// OK state is valid.. now we perform the token exchange
	redirectURL := h.getRedirURL(r)
	tokenRespBody, err := h.getBytesFromSuccessfullPost(h.config.TokenURL,
		url.Values{"redirect_uri": {redirectURL},
			"code":          {authCode},
			"grant_type":    {"authorization_code"},
			"client_id":     {h.config.ClientID},
			"client_secret": {h.config.ClientSecret},
		})
	if err != nil {
		h.params.Logger.Printf("err=%s", err)
		http.Error(w, "bad transaction with openic context ",
			http.StatusInternalServerError)
		return
	}
	var oauth2AccessToken accessToken
	err = json.Unmarshal(tokenRespBody, &oauth2AccessToken)
	if err != nil {
		h.params.Logger.Printf(string(tokenRespBody))
		http.Error(w, "cannot decode oath2 response for token ",
			http.StatusInternalServerError)
		return
	}
	// TODO: tolower
	if oauth2AccessToken.TokenType != "Bearer" ||
		len(oauth2AccessToken.AccessToken) < 1 {
		h.params.Logger.Printf(string(tokenRespBody))
		http.Error(w, "invalid accessToken ", http.StatusInternalServerError)
		return
	}
	h.params.Logger.Debugf(1,
		"oauth2RedirectPathHandler: access token expires: %s\n",
		time.Now().Add(time.Second*time.Duration(oauth2AccessToken.ExpiresIn)))
	// Now we use the access_token (from token exchange) to get userinfo
	userInfoRespBody, err := h.getBytesFromSuccessfullPost(h.config.UserinfoURL,
		url.Values{"access_token": {oauth2AccessToken.AccessToken}})
	if err != nil {
		h.params.Logger.Printf("err=%s", err)
		http.Error(w, "bad transaction with openic context ",
			http.StatusInternalServerError)
		return
	}
	var userInfo openidConnectUserInfo
	err = json.Unmarshal(userInfoRespBody, &userInfo)
	if err != nil {
		h.params.Logger.Printf(string(tokenRespBody))
		http.Error(w, "cannot decode oath2 userinfo token ",
			http.StatusInternalServerError)
		return
	}
	userInfo.Username = getUsernameFromUserinfo(userInfo)
	sort.Strings(userInfo.Groups)
	err = h.setAndStoreAuthCookie(w, r, &userInfo,
		time.Now().Add(authNCookieExpirationDuration))
	if err != nil {
		h.params.Logger.Println(err)
		http.Error(w, "cannot set auth Cookie", http.StatusInternalServerError)
		return
	}
	destinationPath := inboundJWT.ReturnURL
	http.Redirect(w, r, destinationPath, http.StatusFound)
}

func (h *authNHandler) populateEndpoints() error {
	if h.config.AuthURL != "" &&
		h.config.TokenURL != "" &&
		h.config.UserinfoURL != "" {
		return nil
	}
	resp, err := h.netClient.Get(
		h.config.ProviderURL + constants.OpenIDCConfigurationDocumentPath)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		ioutil.ReadAll(resp.Body)
		return errors.New(resp.Status)
	}
	var providerConfig openidConnectProviderConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&providerConfig); err != nil {
		return err
	}
	if h.config.AuthURL == "" {
		h.config.AuthURL = providerConfig.AuthURL
	}
	if h.config.TokenURL == "" {
		h.config.TokenURL = providerConfig.TokenURL
	}
	if h.config.UserinfoURL == "" {
		h.config.UserinfoURL = providerConfig.UserinfoURL
	}
	return nil
}

func (h *authNHandler) verifyAuthnCookie(cookieValue string, issuer string) (
	authInfo, bool, error) {
	if len(cookieValue) < 1 {
		return authInfo{}, false, nil
	}
	inboundJWT := authInfo{}
	tok, err := jwt.ParseSigned(cookieValue)
	if err != nil {
		return inboundJWT, false, nil
	}
	if err := h.JWTClaims(tok, &inboundJWT); err != nil {
		h.params.Logger.Printf("error parsing claims err=%s", err)
		return inboundJWT, false, nil
	}
	subject := "state:" + h.authCookieName
	if inboundJWT.Issuer != issuer || inboundJWT.Subject != subject ||
		inboundJWT.NotBefore > time.Now().Unix() ||
		inboundJWT.Expiration < time.Now().Unix() {
		err = errors.New("invalid JWT values")
		return inboundJWT, false, nil
	}
	authInfo := inboundJWT
	return authInfo, true, nil
}

func (h *authNHandler) getRemoteAuthInfo(w http.ResponseWriter,
	r *http.Request) (*authinfo.AuthInfo, error) {
	h.params.Logger.Debugf(2, "getRemoteAuthInfo: %s query: %s\n",
		r.URL.Path, r.URL.RawQuery)
	// If you have a verified cert, no need for cookies
	if r.TLS != nil && len(r.TLS.VerifiedChains) > 0 {
		authInfo, err := x509util.GetAuthInfo(r.TLS.VerifiedChains[0][0])
		if err == nil {
			return authInfo, nil
		}
		h.params.Logger.Println(err)
	}
	remoteCookie, err := r.Cookie(h.authCookieName)
	if err != nil {
		h.oauth2DoRedirectoToProviderHandler(w, r)
		return nil, fmt.Errorf("error getting cookie: %s: %s",
			h.authCookieName, err)
	}
	authInfo, ok, err := h.verifyAuthnCookie(remoteCookie.Value, r.Host)
	if err != nil {
		return nil, err
	}
	if !ok {
		h.oauth2DoRedirectoToProviderHandler(w, r)
		return nil, errors.New("Cookie not found")
	}
	valid, groups := h.getCachedUserGroups(authInfo.Username)
	if !valid {
		h.oauth2DoRedirectoToProviderHandler(w, r)
		return nil, errors.New("No valid cached group data")
	}
	return &authinfo.AuthInfo{
		Groups:   groups,
		Username: authInfo.Username,
	}, nil
}

func (h *authNHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.params.Logger.Debugf(0, "Inside the handler path=%s query=%s",
		r.URL.Path, r.URL.RawQuery)
	if strings.HasPrefix(r.URL.Path, oauth2redirectPath) {
		h.oauth2RedirectPathHandler(w, r)
		return
	}
	authInfo, err := h.getRemoteAuthInfo(w, r)
	if err != nil {
		h.params.Logger.Debugln(1, err)
		return
	}
	if h.params.AddHeaders {
		r.Header.Set("X-Remote-User", authInfo.Username)
		r.Header.Set("X-Forwarded-User", authInfo.Username)
	}
	if us, ok := w.(usernameSetter); ok {
		us.SetUsername(authInfo.Username)
	}
	h.params.Logger.Debugf(1, "authenticated user: %s for path: %s\n",
		authInfo.Username, r.URL.Path)
	h.params.Handler.ServeHTTP(w,
		r.WithContext(authinfo.ContextWithAuthInfo(r.Context(), *authInfo)))
}
