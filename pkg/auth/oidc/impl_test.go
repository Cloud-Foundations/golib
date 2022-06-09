package oidc

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log/testlogger"
	"github.com/Cloud-Foundations/keymaster/lib/instrumentedwriter"
)

func randomStringGeneration() (string, error) {
	const size = 32
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

type TestHandler struct{}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	okHandler(w, r)
}

func okHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(`ok`))
}

func NewTestHandler() http.Handler {
	return &TestHandler{}
}

type httpTestLogger struct {
	T                *testing.T
	ExpectedUsername string
}

func (l httpTestLogger) Log(record instrumentedwriter.LogRecord) {
	if l.T == nil {
		fmt.Printf("%s -  %s [%s] \"%s %s %s\" %d %d \"%s\"\n",
			record.Ip, record.Username, record.Time, record.Method,
			record.Uri, record.Protocol, record.Status, record.Size,
			record.UserAgent)
		return
	}
	if l.ExpectedUsername != record.Username {
		l.T.Fatal("username doe not match")
	}

}

func checkRequestHandlerCode(req *http.Request, handlerFunc http.HandlerFunc,
	expectedStatus int) (*httptest.ResponseRecorder, error) {
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != expectedStatus {
		errStr := fmt.Sprintf(
			"handler returned wrong status code: got %v want %v",
			status, expectedStatus)
		err := errors.New(errStr)
		return nil, err
	}
	return rr, nil
}

func makeTestAuthNHandler(t *testing.T) *authNHandler {
	return &authNHandler{
		authCookieName:   authCookieNamePrefix,
		cachedUserGroups: make(map[string]expiringGroups),
		params: Params{
			Handler: NewTestHandler(),
			Logger:  testlogger.New(t),
		},
		sharedSecrets: []string{"secret"},
	}
}

func TestOauth2RedirectHandlerSucccess(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		fmt.Fprintln(w, "{\"access_token\": \"6789\", \"token_type\": \"Bearer\",\"username\":\"user\"}")
	}))
	defer ts.Close()
	handler := makeTestAuthNHandler(t)
	handler.config = Config{
		TokenURL:    ts.URL,
		UserinfoURL: ts.URL,
	}
	handler.netClient = ts.Client()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	stateString, err := handler.generateValidStateString(req)
	if err != nil {
		t.Fatal(err)
	}
	v := url.Values{
		"state": {stateString},
		"code":  {"12345"},
	}
	redirReq, err := http.NewRequest("GET", "/?"+v.Encode(), nil)
	rr := httptest.NewRecorder()
	handler.oauth2RedirectPathHandler(rr, redirReq)
	if rr.Code != http.StatusFound {
		t.Fatal("Response should have been a redirect")
	}
	resp := rr.Result()
	if resp.Header.Get("Location") != "/" {
		t.Fatal("Response should have been a redirect to /")
	}
}

func TestGetRemoteUserNameHandler(t *testing.T) {
	handler := makeTestAuthNHandler(t)
	handler.setCachedUserGroups("username", nil, time.Now().Add(time.Minute))
	// Test with no cookies: immediate redirect.
	urlList := []string{"/", "/static/foo"}
	for _, url := range urlList {
		req, err := http.NewRequest("GET", url, nil)
		req.TLS = &tls.ConnectionState{}
		if err != nil {
			t.Fatal(err)
		}
		_, err = checkRequestHandlerCode(req, func(w http.ResponseWriter,
			r *http.Request) {
			_, err := handler.getRemoteAuthInfo(w, r)
			if err == nil {
				t.Fatal("getRemoteAuthInfo should have failed")
			}
		}, http.StatusFound)
		if err != nil {
			t.Fatal(err)
		}
	}
	// Now fail with an unknown cookie.
	unknownCookieReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	unknownCookieReq.TLS = &tls.ConnectionState{}
	cookieVal, err := randomStringGeneration()
	if err != nil {
		t.Fatal(err)
	}
	authCookie := http.Cookie{
		Name:  handler.authCookieName,
		Value: cookieVal,
	}
	unknownCookieReq.AddCookie(&authCookie)
	_, err = checkRequestHandlerCode(unknownCookieReq,
		func(w http.ResponseWriter, r *http.Request) {
			_, err := handler.getRemoteAuthInfo(w, r)
			if err == nil {
				t.Fatal("getRemoteAuthInfo should have failed")
			}
		}, http.StatusFound)
	// Now success with valid cookie.
	goodCookieReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	goodCookieReq.TLS = &tls.ConnectionState{}
	const testUsername = "username"
	validCookie, err := handler.GenValidAuthCookie(&openidConnectUserInfo{
		Username: testUsername,
	})
	if err != nil {
		t.Fatal(err)
	}
	goodCookieReq.AddCookie(validCookie)
	_, err = checkRequestHandlerCode(goodCookieReq, func(w http.ResponseWriter,
		r *http.Request) {
		r.Host = "localhost"
		authInfo, err := handler.getRemoteAuthInfo(w, r)
		if err != nil {
			t.Fatalf("getRemoteAuthInfo should NOT have failed, err: %s", err)
		}
		if authInfo.Username != testUsername {
			t.Fatal("getRemoteAuthInfo.Username does NOT match")
		}
	}, http.StatusOK)
	// Now failure with an expired Cookie.
	expiredCookieReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	expiredCookieReq.TLS = &tls.ConnectionState{}
	expiredCookie, err := handler.genValidAuthCookieExpiration(
		&openidConnectUserInfo{Username: testUsername},
		time.Now().Add(-10*time.Second), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	expiredCookieReq.AddCookie(expiredCookie)
	_, err = checkRequestHandlerCode(expiredCookieReq,
		func(w http.ResponseWriter, r *http.Request) {
			_, err := handler.getRemoteAuthInfo(w, r)
			if err == nil {
				t.Fatal("getRemoteAuthInfo should have failed")
			}
		}, http.StatusFound)
}

func TestAutnnHandlerValid(t *testing.T) {
	handler := makeTestAuthNHandler(t)
	handler.setCachedUserGroups("username", nil, time.Now().Add(time.Minute))
	// Test success with valid cookie.
	goodCookieReq, err := http.NewRequest("GET", "/", nil)
	goodCookieReq.TLS = &tls.ConnectionState{}
	if err != nil {
		t.Fatal(err)
	}
	const testUsername = "username"
	validCookie, err := handler.GenValidAuthCookie(&openidConnectUserInfo{
		Username: testUsername,
	})
	if err != nil {
		t.Fatal(err)
	}
	goodCookieReq.AddCookie(validCookie)
	goodCookieReq.Host = "localhost"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, goodCookieReq)
	if rr.Code != http.StatusOK {
		t.Fatal("Authentication Failed")
	}
	// Now we should get a redirect if reaching the redirecturl.
	oauth2redirectReq, err := http.NewRequest("GET", oauth2redirectPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, oauth2redirectReq)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatal("Ouath2 redirect did not failed")
	}
	// Now we test with a wrapped handler to ensure username is set.
	l := httpTestLogger{
		ExpectedUsername: testUsername,
		T:                t,
	}
	wrappedHandler := instrumentedwriter.NewLoggingHandler(handler, l)
	rr3 := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr3, goodCookieReq)
	if rr3.Code != http.StatusOK {
		t.Fatal("Authentication Failed")
	}
	// Finally we put a bad cookie.
	badCookie := validCookie
	badCookie.Value = "Foo"
	badCookieReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	badCookieReq.TLS = &tls.ConnectionState{}
	badCookieReq.AddCookie(validCookie)
	badCookieReq.Host = "localhost"
	rr4 := httptest.NewRecorder()
	handler.ServeHTTP(rr4, badCookieReq)
	if rr4.Code != http.StatusFound {
		t.Fatal("Bad cookie should redirect")
	}
}
