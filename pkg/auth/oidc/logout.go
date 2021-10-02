package oidc

import (
	"net/http"
	"time"
)

const logoutPage = `<title>Logout Page</title>
<body>
<center>
<h1>You have logged out</h1>
<h2><a href="/">Login</a></h2><br>
</center>
</body>`

func defaultLogoutHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(logoutPage))
}

func (h *authNHandler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.authCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(time.Minute),
		HttpOnly: true,
		Secure:   true,
	})
	h.params.LogoutHandler(w, r)
}
