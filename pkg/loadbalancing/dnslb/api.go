/*
Package dnslb implements a server-side DNS Load Balancer for a cluster of
servers (typically Web application servers).

Each server instance will add it's IP address in an A record for the specified
FQDN to the DNS system and will monitor the health of other servers
(using TCP/TLS probes) and will remove failed servers from the pool of A
records.

Clients are expected to use Round-Robin DNS to randomly select which server to
connect to as well as failover to other servers if the chosen server fails
(until the failed server is removed from DNS, after which clients will no longer
"see" the server).

Using the dnslb package, a highly available server cluster may be established
without the need for an external load balancer (which is a single point of
failure and increases cost and complexity). The only dependency is a functioning
DNS system, which almost always is an essential service anyway; thus, no extra
dependency is introduced. Furthermore, no separate DNS record management is
required as the servers self-register.

If there is a network break between servers their A records will be added and
removed periodically until the network break is fixed.

The config sub-package allows for easy configuration and selection of DNS
provider backends such as AWS Route53.
*/
package dnslb

import (
	"math/rand"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

type Config struct {
	CheckInterval   time.Duration `yaml:"check_interval"` // Minumum: 5s.
	DoTLS           bool          `yaml:"do_tls"`
	FQDN            string        `yaml:"fqdn"`
	MaximumFailures uint          `yaml:"maximum_failures"` // Default: 60.
	MinimumFailures uint          `yaml:"minimum_failures"` // Default:  3.
	TcpPort         uint16        `yaml:"tcp_port"`
}

// Destroyer implements the Destroy method, used to destroy instances.
type Destroyer interface {
	Destroy(ips map[string]struct{}) error
}

type LoadBalancer struct {
	config   Config
	failures map[string]uint // Key: IP, value: failure count.
	myIP     string          // TODO(rgooch): add IPv6 support.
	p        Params
	rand     *rand.Rand
}

type Params struct {
	Destroyer        Destroyer
	Logger           log.DebugLogger
	RecordReadWriter RecordReadWriter
	RegionFilter     RegionFilter
}

// RecordReadWriter implements a DNS record reader and writer. It is used to
// plugin the underlying DNS provider.
type RecordReadWriter interface {
	DeleteRecords(fqdn, recType string) error
	ReadRecords(fqdn, recType string) ([]string, time.Duration, error)
	WriteRecords(fqdn, recType string, recs []string, ttl time.Duration) error
}

// RegionFilter implements the Filter method, which is used to restrict DNS
// changes and instance destruction to the same region (this avoids network
// partition problems).
type RegionFilter interface {
	Filter(ips map[string]struct{}) (map[string]struct{}, error)
}

// New creates a *LoadBalancer using the provided configuration and back-end
// DNS provider. This will launch a goroutine to perform periodic health checks
// for the peer servers and to self register.
func New(config Config, params Params) (*LoadBalancer, error) {
	return newLoadBalancer(config, params)
}

// Block will block a server instance with the specified IP address from
// adding itself to DNS for the specified time or until a message is received on
// cancelChannel.
func Block(config Config, params Params, ip string, duration time.Duration,
	cancelChannel <-chan struct{}, logger log.DebugLogger) error {
	return block(config, params, ip, duration, cancelChannel, logger)
}

// RollingReplace will use the provided configuration and will roll through all
// server instances in the specified region triggering replacements by removing
// each server from DNS, destroying it and waiting for (some other mechanism) to
// create a working replacement before continuing to the next server.
func RollingReplace(config Config, params Params, region string,
	logger log.DebugLogger) error {
	return rollingReplace(config, params, region, logger)
}
