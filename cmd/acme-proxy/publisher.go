package main

import (
	"io/ioutil"
	"net"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/golib/pkg/constants"
)

const (
	maxResponses = 100
	maxSize      = 1 << 16
)

func (proxy *acmeProxy) setupPublisher() error {
	proxy.ipMap = make(map[string]*responsesType)
	html.HandleFunc(constants.AcmeProxyCleanupResponses, proxy.cleanupHandler)
	html.HandleFunc(constants.AcmeProxyRecordResponse, proxy.recordHandler)
	return nil
}

func (proxy *acmeProxy) getResponse(hostPort, path string) []byte {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		proxy.logger.Println(err)
		return nil
	}
	proxy.rwMutex.RLock()
	defer proxy.rwMutex.RUnlock()
	for _, ip := range ips {
		if responses := proxy.ipMap[ip]; responses != nil {
			if data := responses.pathMap[path]; len(data) > 0 {
				return data
			}
		}
	}
	return nil
}

func (proxy *acmeProxy) cleanupHandler(w http.ResponseWriter,
	req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, "Use POST", http.StatusMethodNotAllowed)
		return
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		http.Error(w, "Cannot split host:port", http.StatusInternalServerError)
		proxy.logger.Println(err)
	}
	proxy.rwMutex.Lock()
	defer proxy.rwMutex.Unlock()
	delete(proxy.ipMap, host)
	proxy.logger.Debugf(0, "cleaned up for: %s\n", host)
}

func (proxy *acmeProxy) recordHandler(w http.ResponseWriter,
	req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, "Use POST", http.StatusMethodNotAllowed)
		return
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		http.Error(w, "Cannot split host:port", http.StatusInternalServerError)
		proxy.logger.Println(err)
		return
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		proxy.logger.Println(err)
		return
	}
	if len(data) > maxSize {
		http.Error(w, "Too much data", http.StatusNotAcceptable)
		return
	}
	proxy.rwMutex.Lock()
	defer proxy.rwMutex.Unlock()
	responses := proxy.ipMap[host]
	if responses == nil {
		responses = &responsesType{make(map[string][]byte)}
		proxy.ipMap[host] = responses
	} else if len(responses.pathMap) >= maxResponses {
		http.Error(w, "Too much data", http.StatusTooManyRequests)
		return
	}
	if len(responses.pathMap[req.URL.RawQuery]) > 0 {
		http.Error(w, "Duplicate path", http.StatusConflict)
		return
	}
	responses.pathMap[req.URL.RawQuery] = data
	w.WriteHeader(http.StatusOK)
	proxy.logger.Printf("%s: recorded for path: %s\n", host, req.URL.RawQuery)
}
