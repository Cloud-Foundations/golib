package watchdog

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

type Config struct {
	ArmTime       time.Duration `yaml:"arm_time"`       // Def: 10s, min: 1s.
	CheckInterval time.Duration `yaml:"check_interval"` // Def: 5s, min: 100ms.
	DoTLS         bool          `yaml:"do_tls"`
	ExitTime      time.Duration `yaml:"exit_time"` // Def: 15s, min: check*2.
	TcpPort       uint16        `yaml:"tcp_port"`
}

type Watchdog struct {
	addr   string
	c      Config
	logger log.DebugLogger
}

func New(config Config, logger log.DebugLogger) (*Watchdog, error) {
	return newWatchdog(config, logger)
}

func (config *Config) SetDefaults() {
	config.setDefaults()
}
