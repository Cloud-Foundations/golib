package http_proxy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Cloud-Foundations/golib/pkg/constants"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

func newResponder(acmeProxy string,
	logger log.DebugLogger) (*Responder, error) {
	return &Responder{acmeProxy, logger}, nil
}

func (r *Responder) cleanup() {
	resp, err := http.DefaultClient.Post(
		"http://"+r.acmeProxy+constants.AcmeProxyCleanupResponses, "", nil)
	if err != nil {
		r.logger.Println(err)
	} else if resp.StatusCode != http.StatusOK {
		r.logger.Println(resp.Status)
	}
}

func (r *Responder) respond(key, value string) error {
	if !strings.HasPrefix(key, constants.AcmePath) {
		return errors.New("not an ACME challenge response")
	}
	url := "http://" + r.acmeProxy + constants.AcmeProxyRecordResponse + "?" +
		key
	resp, err := http.DefaultClient.Post(url, "", strings.NewReader(value))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s\n", url, resp.Status)
	}
	return nil
}
