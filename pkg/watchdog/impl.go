package watchdog

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

func (config *Config) setDefaults() {
	config.ArmTime = time.Second * 10
	config.CheckInterval = time.Second * 5
	config.ExitTime = time.Second * 15
}

func newWatchdog(config Config, logger log.DebugLogger) (*Watchdog, error) {
	if config.TcpPort < 1 {
		return nil, errors.New("no TCP port number specified")
	}
	if config.ArmTime < time.Second {
		config.ArmTime = time.Second // Minimum.
	}
	if config.CheckInterval < time.Millisecond*100 {
		config.CheckInterval = time.Millisecond * 100 // Minimum.
	}
	if config.ExitTime < config.CheckInterval*2 {
		config.ExitTime = config.CheckInterval * 2 // Minimum.
	}
	w := &Watchdog{
		addr:   fmt.Sprintf(":%d", config.TcpPort),
		c:      config,
		logger: logger,
	}
	go w.enter()
	return w, nil
}

func (w *Watchdog) check() error {
	deadline := time.Now().Add(w.c.CheckInterval)
	conn, err := net.DialTimeout("tcp", w.addr, w.c.CheckInterval)
	if err != nil {
		return err
	}
	defer conn.Close()
	if w.c.DoTLS {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
		tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
		defer tlsConn.Close()
		if err := tlsConn.Handshake(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Watchdog) enter() {
	w.logger.Debugln(0, "watchdog: waiting to arm")
	w.waitToArm()
	if w.c.DoTLS {
		w.logger.Printf("watchdog(%s/%s): armed, watching TCP/TLS port: %d\n",
			w.c.CheckInterval, w.c.ExitTime, w.c.TcpPort)
	} else {
		w.logger.Printf("watchdog(%s/%s): armed, watching TCP port: %d\n",
			w.c.CheckInterval, w.c.ExitTime, w.c.TcpPort)
	}
	w.watch()
	w.logger.Fatalln("watchdog triggered")
	panic("watchdog did not exit")
}

func (w *Watchdog) watch() {
	lastSuccess := time.Now()
	lastWasBad := false
	for time.Since(lastSuccess) < w.c.ExitTime {
		wakeAt := time.Now().Add(w.c.CheckInterval - time.Millisecond*10)
		if err := w.check(); err != nil {
			w.logger.Printf("watchdog: %s\n", err)
			lastWasBad = true
		} else {
			lastSuccess = time.Now()
			if lastWasBad {
				lastWasBad = false
				w.logger.Println("watchdog: check succeeded again")
			}
		}
		time.Sleep(time.Until(wakeAt))
	}
}

func (w *Watchdog) waitToArm() {
	lastFailure := time.Now()
	armAt := lastFailure.Add(w.c.ArmTime)
	for time.Until(armAt) > 0 {
		wakeAt := time.Now().Add(w.c.CheckInterval - time.Millisecond*10)
		if err := w.check(); err != nil {
			lastFailure = time.Now()
			armAt = lastFailure.Add(w.c.ArmTime)
			w.logger.Println(err)
		} else if armAt.Before(wakeAt) {
			wakeAt = armAt // Don't sleep the normal interval.
		}
		time.Sleep(time.Until(wakeAt))
	}
}
