package ldaputil

import (
	"crypto/x509"
	"net/url"
	"time"
)

func CheckLDAPConnection(u url.URL, timeout time.Duration,
	rootCAs *x509.CertPool) error {
	return checkLDAPConnection(u, timeout, rootCAs)
}

func CheckLDAPUserPassword(u url.URL, bindDN string, bindPassword string,
	timeout time.Duration, rootCAs *x509.CertPool) (bool, error) {
	return checkLDAPUserPassword(u, bindDN, bindPassword, timeout, rootCAs)
}

func GetLDAPUserGroups(u url.URL, bindDN string, bindPassword string,
	timeout time.Duration, rootCAs *x509.CertPool, username string,
	UserSearchBaseDNs []string, UserSearchFilter string,
	GroupSearchBaseDNs []string, GroupSearchFilter string) ([]string, error) {
	return getLDAPUserGroups(u, bindDN, bindPassword, timeout, rootCAs,
		username, UserSearchBaseDNs, UserSearchFilter,
		GroupSearchBaseDNs, GroupSearchFilter)
}

func GetLDAPGroupUsers(u url.URL, bindDN string, bindPassword string,
	timeout time.Duration, rootCAs *x509.CertPool,
	userAttribute string,
	groupName string,
	UserSearchBaseDNs []string,
	GroupSearchBaseDNs []string) ([]string, error) {
	return getLDAPGroupUsers(u, bindDN, bindPassword, timeout, rootCAs,
		userAttribute, groupName,
		UserSearchBaseDNs, GroupSearchBaseDNs)
}

func ParseLDAPURL(ldapUrl string) (*url.URL, error) {
	return parseLDAPURL(ldapUrl)
}
