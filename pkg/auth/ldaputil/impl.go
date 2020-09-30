package ldaputil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gopkg.in/ldap.v2"
)

func checkLDAPConnection(u url.URL, timeout time.Duration,
	rootCAs *x509.CertPool) error {
	conn, _, err := getLDAPConnection(u, timeout, rootCAs)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetTimeout(timeout)
	conn.Start()
	return nil
}

func checkLDAPUserPassword(u url.URL, bindDN string, bindPassword string,
	timeout time.Duration, rootCAs *x509.CertPool) (bool, error) {
	conn, server, err := getLDAPConnection(u, timeout, rootCAs)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	conn.SetTimeout(timeout)
	conn.Start()
	err = conn.Bind(bindDN, bindPassword)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid Credentials") {
			return false, nil
		}
		return false,
			fmt.Errorf("Bind failure for server:%s bindDN:'%s' (%s)",
				server, bindDN, err)
	}
	return true, nil
}

func extractCNFromDNString(input []string) (output []string, err error) {
	re := regexp.MustCompile("(?i)^cn=([^,]+),.*")
	for _, dn := range input {
		matches := re.FindStringSubmatch(dn)
		if len(matches) == 2 {
			output = append(output, matches[1])
		} else {
			output = append(output, dn)
		}
	}
	return output, nil
}

func getLDAPConnection(u url.URL, timeout time.Duration,
	rootCAs *x509.CertPool) (*ldap.Conn, string, error) {
	if u.Scheme != "ldaps" {
		err := errors.New("Invalid ldap scheme (we only support ldaps")
		return nil, "", err
	}
	// hostnamePort := server + ":636"
	serverPort := strings.Split(u.Host, ":")
	port := "636"
	if len(serverPort) == 2 {
		port = serverPort[1]
	}
	server := serverPort[0]
	hostnamePort := server + ":" + port
	start := time.Now()
	tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp",
		hostnamePort, &tls.Config{ServerName: server, RootCAs: rootCAs})
	if err != nil {
		errorTime := time.Since(start).Seconds() * 1000
		return nil, "", fmt.Errorf("connction failure for:%s (%s)(time(ms)=%v)",
			server, err.Error(), errorTime)
	}
	// Closing the LDAP connection will close the TLS connection.
	conn := ldap.NewConn(tlsConn, true)
	return conn, server, nil
}

func getLDAPUserGroups(u url.URL, bindDN string, bindPassword string,
	timeout time.Duration, rootCAs *x509.CertPool,
	username string,
	UserSearchBaseDNs []string, UserSearchFilter string,
	GroupSearchBaseDNs []string, GroupSearchFilter string) ([]string, error) {
	conn, _, err := getLDAPConnection(u, timeout, rootCAs)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetTimeout(timeout)
	conn.Start()
	err = conn.Bind(bindDN, bindPassword)
	if err != nil {
		return nil, err
	}
	rfcGroups, err := getUserGroupsRFC2307(conn, GroupSearchBaseDNs,
		GroupSearchFilter, username)
	if err != nil {
		return nil, err
	}
	memberGroups, err := getUserGroupsRFC2307bis(conn, UserSearchBaseDNs,
		UserSearchFilter, username)
	if err != nil {
		return nil, err
	}
	groupMap := make(map[string]struct{})
	for _, group := range rfcGroups {
		groupMap[group] = struct{}{}
	}
	for _, group := range memberGroups {
		groupMap[group] = struct{}{}
	}
	var userGroups []string
	for group := range groupMap {
		userGroups = append(userGroups, group)
	}
	return userGroups, nil
}

func getUserDNAndSimpleGroups(conn *ldap.Conn, UserSearchBaseDNs []string,
	UserSearchFilter string, username string) (string, []string, error) {
	for _, searchDN := range UserSearchBaseDNs {
		searchRequest := ldap.NewSearchRequest(
			searchDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf(UserSearchFilter, username),
			[]string{"dn", "memberOf"},
			nil,
		)
		sr, err := conn.Search(searchRequest)
		if err != nil {
			return "", nil, err
		}
		if len(sr.Entries) != 1 {
			continue
		}
		userDN := sr.Entries[0].DN
		userGroups := sr.Entries[0].GetAttributeValues("memberOf")
		return userDN, userGroups, nil
	}
	return "", nil, nil
}

func getUserGroupsRFC2307bis(conn *ldap.Conn, UserSearchBaseDNs []string,
	UserSearchFilter string, username string) ([]string, error) {
	dn, groupDNs, err := getUserDNAndSimpleGroups(conn, UserSearchBaseDNs,
		UserSearchFilter, username)
	if err != nil {
		return nil, err
	}
	if dn == "" {
		return nil,
			errors.New("User does not exist or too many entries returned")
	}
	groupCNs, err := extractCNFromDNString(groupDNs)
	if err != nil {
		return nil, err
	}
	return groupCNs, nil
}

func getUserGroupsRFC2307(conn *ldap.Conn, GroupSearchBaseDNs []string,
	groupSearchFilter string,
	username string) (userGroups []string, err error) {
	for _, searchDN := range GroupSearchBaseDNs {
		searchRequest := ldap.NewSearchRequest(
			searchDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf(groupSearchFilter, username),
			[]string{"cn"}, nil)
		sr, err := conn.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("error on search request: %s", err)
		}
		for _, entry := range sr.Entries {
			userGroups = append(userGroups, entry.GetAttributeValues("cn")...)
		}
	}
	return userGroups, nil
}

func parseLDAPURL(ldapUrl string) (*url.URL, error) {
	u, err := url.Parse(ldapUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ldaps" {
		return nil, errors.New("Invalid ldap scheme (we only support ldaps")
	}
	return u, nil
}
