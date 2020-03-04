package http

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

const acmePath = "/.well-known/acme-challenge"

func createListener(portNum uint16) (net.Listener, error) {
	return net.Listen("tcp", ":"+strconv.FormatInt(int64(portNum), 10))
}

func createRedirectServer(portNum uint16, logger log.DebugLogger) error {
	listener, err := createListener(portNum)
	if err != nil {
		return err
	}
	go runServer(listener, &RedirectHandler{}, logger)
	return nil
}

func runServer(listener net.Listener, handler http.Handler, logger log.Logger) {
	if err := http.Serve(listener, handler); err != nil {
		logger.Println(err)
	}
}

type rejectingListener struct {
	listener  net.Listener
	responder *Responder
}

func newHandler(fallback http.Handler,
	logger log.DebugLogger) (*Responder, error) {
	return &Responder{
		fallback:  fallback,
		logger:    logger,
		responses: make(map[string]string),
	}, nil
}

func newServer(portNum uint16, fallback http.Handler,
	logger log.DebugLogger) (*Responder, error) {
	responder, err := newHandler(fallback, logger)
	if err != nil {
		return nil, err
	}
	listener, err := createListener(portNum)
	if err != nil {
		return nil, err
	}
	if fallback == nil {
		listener = &rejectingListener{listener: listener, responder: responder}
	}
	go runServer(listener, responder, logger)
	return responder, nil
}

func stripPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}

func (*RedirectHandler) serveHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "HEAD" {
		http.Error(w, "Use HTTPS", http.StatusBadRequest)
		return
	}
	target := "https://" + stripPort(req.Host) + req.URL.RequestURI()
	http.Redirect(w, req, target, http.StatusFound)
}

func (l *rejectingListener) Accept() (net.Conn, error) {
	for {
		if conn, err := l.listener.Accept(); err != nil {
			return nil, err
		} else {
			l.responder.rwMutex.RLock()
			haveResponses := len(l.responder.responses) > 0
			l.responder.rwMutex.RUnlock()
			if haveResponses {
				return conn, nil
			}
			l.responder.logger.Debugf(2, "closing connection from: %s\n",
				conn.RemoteAddr())
			conn.Close()
		}
	}
}

func (l *rejectingListener) Close() error {
	return l.listener.Close()
}

func (l *rejectingListener) Addr() net.Addr {
	return l.listener.Addr()
}

func (r *Responder) cleanup() {
	r.rwMutex.Lock()
	r.responses = make(map[string]string)
	r.rwMutex.Unlock()
}

func (r *Responder) serveHTTP(w http.ResponseWriter, req *http.Request) {
	r.logger.Debugf(1, "source: %s, method: %s, path: %s\n",
		req.RemoteAddr, req.Method, req.URL.Path)
	if !strings.HasPrefix(req.URL.Path, acmePath) {
		if r.fallback == nil {
			http.Error(w, "not an ACME challenge", http.StatusNotFound)
		} else {
			r.fallback.ServeHTTP(w, req)
		}
		return
	}
	response := r.getResponse(req.URL.Path)
	if response == "" {
		http.Error(w, "no token for path", http.StatusNotFound)
		r.logger.Debugf(0, "no token for path: %s\n", req.URL.Path)
		return
	}
	w.Write([]byte(response))
}

func (r *Responder) getResponse(key string) string {
	r.rwMutex.RLock()
	response, ok := r.responses[key]
	r.rwMutex.RUnlock()
	if !ok {
		return ""
	}
	return response
}

func (r *Responder) respond(key, value string) error {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()
	r.responses[key] = value
	return nil
}
