module github.com/Cloud-Foundations/golib

go 1.15

replace github.com/go-fsnotify/fsnotify v0.0.0-20180321022601-755488143dae => github.com/fsnotify/fsnotify v1.4.9

require (
	github.com/Cloud-Foundations/Dominator v0.0.0-20210524064856-a7256858e533
	github.com/Cloud-Foundations/tricorder v0.0.0-20191102180116-cf6bbf6d0168
	github.com/aws/aws-sdk-go v1.40.22
	github.com/go-fsnotify/fsnotify v0.0.0-20180321022601-755488143dae // indirect
	github.com/prometheus/client_golang v1.11.0
	github.com/vjeantet/ldapserver v1.0.1
	golang.org/x/crypto v0.0.0-20210813211128-0a44fdfbc16e
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/fsnotify/fsnotify.v0 v0.9.3 // indirect
	gopkg.in/ldap.v2 v2.5.1
	gopkg.in/yaml.v2 v2.4.0
)
