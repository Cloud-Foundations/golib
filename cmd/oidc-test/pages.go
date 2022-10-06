package main

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/golib/pkg/auth/oidc"
)

func genericHandler(w http.ResponseWriter, req *http.Request, pageName string) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintf(writer, "<title>oidc-test %s</title>\n", pageName)
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintf(writer, "<h1>oidc-test %s</h1>\n", pageName)
	fmt.Fprintln(writer, "</center>")
	oidc.GetAuthInfoFromRequest(req).Write(writer, "", "&nbsp&nbsp", "<br>")
	fmt.Fprintln(writer, "<p>")
	fmt.Fprintln(writer,
		`<form enctype="application/x-www-form-urlencoded" action="/logout" method="post">`)
	fmt.Fprintln(writer, `  <input type="submit" value="Logout">`)
	fmt.Fprintln(writer, "</form>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	genericHandler(w, req, "root page")
}

func page0Handler(w http.ResponseWriter, req *http.Request) {
	genericHandler(w, req, "page0")
}

func page1Handler(w http.ResponseWriter, req *http.Request) {
	genericHandler(w, req, "page1")
}
