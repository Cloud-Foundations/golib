package x509util

import (
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"strings"

	"github.com/Cloud-Foundations/golib/pkg/auth/authinfo"
	"github.com/Cloud-Foundations/golib/pkg/constants"
)

func getAuthInfo(cert *x509.Certificate) (*authinfo.AuthInfo, error) {
	ai := &authinfo.AuthInfo{Username: cert.Subject.CommonName}
	groups, err := getList(cert, constants.GroupListOID)
	if err != nil {
		return nil, fmt.Errorf("error getting group list: %s", err)
	}
	ai.Groups = authinfo.MapToList(groups)
	methods, err := getList(cert, constants.PermittedMethodListOID)
	if err != nil {
		return nil, fmt.Errorf("error getting method list: %s", err)
	}
	for method := range methods {
		if strings.Count(method, ".") != 1 {
			return nil, fmt.Errorf("bad method line: \"%s\"", method)
		}
	}
	ai.PermittedMethods = authinfo.MapToList(methods)
	return ai, nil
}

func getList(cert *x509.Certificate, oid string) (map[string]struct{}, error) {
	list := make(map[string]struct{})
	for _, extension := range cert.Extensions {
		if extension.Id.String() != oid {
			continue
		}
		var lines []string
		rest, err := asn1.Unmarshal(extension.Value, &lines)
		if err != nil {
			return nil, err
		}
		if len(rest) > 0 {
			return nil, fmt.Errorf("%d extra bytes in extension", len(rest))
		}
		for _, line := range lines {
			list[line] = struct{}{}
		}
		return list, nil
	}
	return list, nil
}
