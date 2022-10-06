package constants

const (
	OpenIDCTestStatusPort = 6922
	CertmanagerPortNumber = 6940
	AcmeProxyPortNumber   = 6941

	AcmePath                  = "/.well-known/acme-challenge"
	AcmeProxyCleanupResponses = "/api/responses/cleanup"
	AcmeProxyRecordResponse   = "/api/responses/recordOne"

	OpenIDCConfigurationDocumentPath = "/.well-known/openid-configuration"

	// Copied from github.com/Cloud-Foundations/Dominator/constants
	AssignedOIDBase        = "1.3.6.1.4.1.9586.100.7"
	PermittedMethodListOID = AssignedOIDBase + ".1"
	GroupListOID           = AssignedOIDBase + ".2"
)
