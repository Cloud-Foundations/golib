package oidc

import (
	"net/http"
	"time"
)

const logoutPage = `<title>Logout Page</title>
<body>
<center>
<h1>You are logged out</h1>
<form enctype="application/x-www-form-urlencoded" action="/login" method="post">
<input type="submit" value="Login" />
</center>
</body>`

func defaultLogoutHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(logoutPage))
}

func (h *authNHandler) logout(w http.ResponseWriter, r *http.Request) {
	authCookie, err := r.Cookie(h.authCookieName)
	if authCookie != nil && err == nil {
		authInfo, ok, err := h.verifyAuthnCookie(authCookie.Value, r.Host)
		if ok && err == nil {
			h.mutex.Lock()
			delete(h.cachedUserGroups, authInfo.Username)
			h.mutex.Unlock()
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.authCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(time.Minute),
		HttpOnly: true,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     loginCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(time.Minute),
		HttpOnly: true,
		Secure:   true,
	})
	h.params.LogoutHandler(w, r)
}
