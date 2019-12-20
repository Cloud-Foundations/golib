package ldap

import (
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/keymaster/lib/authutil"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dependencyLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "cloudgate_ldap_userinfo_check_duration_seconds",
			Help:       "LDAP Dependency latency",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"target"},
	)
	userinfoLDAPAttempt = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cloudgate_ldap_userinfo_attempt_counter",
			Help: "Attempts to get userinfo from ldap",
		},
	)
	userinfoLDAPSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cloudgate_ldap_userinfo_success_counter",
			Help: "Success count when getting userinfo from ldap",
		},
	)
)

func init() {
	prometheus.MustRegister(dependencyLatency)
	prometheus.MustRegister(userinfoLDAPAttempt)
	prometheus.MustRegister(userinfoLDAPSuccess)
}

func newUserInfo(urlList []string, bindUsername string, bindPassword string,
	groupSearchFilter string, groupSearchBaseDNs []string,
	userSearchFilter string, userSearchBaseDNs []string,
	timeoutSecs uint, rootCAs *x509.CertPool, logger log.DebugLogger) (
	*UserInfo, error) {
	userinfo := &UserInfo{
		bindUsername:       bindUsername,
		bindPassword:       bindPassword,
		groupSearchFilter:  groupSearchFilter,
		groupSearchBaseDNs: groupSearchBaseDNs,
		userSearchFilter:   userSearchFilter,
		userSearchBaseDNs:  userSearchBaseDNs,
		timeoutSecs:        timeoutSecs,
		rootCAs:            rootCAs,
		logger:             logger,
	}
	for _, stringURL := range urlList {
		url, err := authutil.ParseLDAPURL(stringURL)
		if err != nil {
			return nil, err
		}
		userinfo.ldapURLs = append(userinfo.ldapURLs, url)
	}
	return userinfo, nil
}

func (uinfo *UserInfo) getUserGroups(username string) ([]string, error) {
	ldapSuccess := false
	var groups []string
	var err error
	userinfoLDAPAttempt.Inc()
	for _, ldapUrl := range uinfo.ldapURLs {
		targetName := strings.ToLower(ldapUrl.Hostname())
		startTime := time.Now()
		groups, err = authutil.GetLDAPUserGroups(*ldapUrl,
			uinfo.bindUsername, uinfo.bindPassword,
			uinfo.timeoutSecs, uinfo.rootCAs,
			username,
			uinfo.userSearchBaseDNs, uinfo.userSearchFilter,
			uinfo.groupSearchBaseDNs, uinfo.groupSearchFilter)
		if err != nil {
			continue
		}
		dependencyLatency.WithLabelValues(targetName).Observe(time.Now().Sub(startTime).Seconds())
		ldapSuccess = true
		break
	}
	if !ldapSuccess {
		return nil, fmt.Errorf("Could not contact any configured LDAP endpoint. Last Err: %s", err)
	}
	userinfoLDAPSuccess.Inc()
	uinfo.logger.Debugf(2, "groups=%+v", groups)
	return groups, nil
}
