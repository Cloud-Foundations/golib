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
	"net"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

type Config struct {
	CheckInterval   time.Duration `yaml:"check_interval"` // Minumum: 5s.
	DoTLS           bool          `yaml:"do_tls"`
	FQDN            string        `yaml:"fqdn"`
	MinimumFailures uint          `yaml:"minimum_failures"` // Default: 3.
	TcpPort         uint16        `yaml:"tcp_port"`
}

// RecordReadWriter implements a DNS record reader and writer. It is used to
// plugin the underlying DNS provider.
type RecordReadWriter interface {
	ReadRecord(fqdn string) ([]net.IP, error)
	WriteRecord(fqdn string, ips []net.IP, ttl time.Duration) error
}

type LoadBalancer struct {
	backend    RecordReadWriter
	config     Config
	failures   map[string]uint // Key: IP, value: failure count.
	myIP       net.IP
	myStringIP string
	logger     log.DebugLogger
	rand       *rand.Rand
}

// New creates a *LoadBalancer using the provided configuration and back-end
// DNS provider. This will launch a goroutine to perform periodic health checks
// for the peer servers and to self register.
func New(config Config, backend RecordReadWriter,
	logger log.DebugLogger) (*LoadBalancer, error) {
	return newLoadBalancer(config, backend, logger)
}
