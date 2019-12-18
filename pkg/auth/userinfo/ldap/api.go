package ldap

import (
	"crypto/x509"
	"net/url"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

type UserInfo struct {
	ldapURLs           []*url.URL
	bindUsername       string
	bindPassword       string
	groupSearchFilter  string
	groupSearchBaseDNs []string
	userSearchFilter   string
	userSearchBaseDNs  []string
	timeoutSecs        uint
	rootCAs            *x509.CertPool
	logger             log.DebugLogger
}

func New(urlList []string, bindUsername string, bindPassword string,
	groupSearchFilter string, groupSearchBaseDNs []string,
	userSearchFilter string, userSearchBaseDNs []string,
	timeoutSecs uint, rootCAs *x509.CertPool, logger log.DebugLogger) (
	*UserInfo, error) {
	return newUserInfo(urlList, bindUsername, bindPassword,
		groupSearchFilter, groupSearchBaseDNs,
		userSearchFilter, userSearchBaseDNs, timeoutSecs, rootCAs, logger)
}

func (uinfo *UserInfo) GetUserGroups(username string) ([]string, error) {
	return uinfo.getUserGroups(username)
}
