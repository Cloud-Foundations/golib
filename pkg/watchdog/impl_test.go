package watchdog

import (
	"net"
	"testing"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/Cloud-Foundations/golib/pkg/log/testlogger"
)

func acceptor(listener net.Listener, logger log.DebugLogger) {
	for {
		if conn, err := listener.Accept(); err != nil {
			logger.Println(err)
		} else {
			conn.Close()
		}
	}
}

func createTestWatchdog(accept bool,
	logger log.DebugLogger) (*Watchdog, error) {
	if listener, err := net.Listen("tcp", ":"); err != nil {
		return nil, err
	} else {
		w := &Watchdog{
			addr: listener.Addr().String(),
			c: Config{
				ArmTime:       time.Millisecond * 200,
				CheckInterval: time.Millisecond * 100,
				ExitTime:      time.Millisecond * 200,
			},
			logger: logger,
		}
		if accept {
			go acceptor(listener, logger)
		} else {
			listener.Close()
		}
		return w, nil
	}
}

func TestNoTcpPort(t *testing.T) {
	logger := testlogger.New(t)
	_, err := New(Config{}, logger)
	if err == nil {
		t.Error("expected failure without TCP port")
	}
}

func TestArmingFails(t *testing.T) {
	logger := testlogger.New(t)
	w, err := createTestWatchdog(false, logger)
	if err != nil {
		t.Fatal(err)
	}
	armed := make(chan struct{})
	go func() {
		w.waitToArm()
		armed <- struct{}{}
	}()
	timer := time.NewTimer(time.Millisecond * 500)
	select {
	case <-armed:
		t.Error("watchdog was armed")
	case <-timer.C:
	}
}

func TestArmingWorks(t *testing.T) {
	logger := testlogger.New(t)
	w, err := createTestWatchdog(true, logger)
	if err != nil {
		t.Fatal(err)
	}
	armed := make(chan struct{})
	go func() {
		w.waitToArm()
		armed <- struct{}{}
	}()
	timer := time.NewTimer(time.Millisecond * 500)
	select {
	case <-armed:
	case <-timer.C:
		t.Error("watchdog failed to arm in time")
	}
}

func TestWatchingFails(t *testing.T) {
	logger := testlogger.New(t)
	w, err := createTestWatchdog(false, logger)
	if err != nil {
		t.Fatal(err)
	}
	failed := make(chan struct{})
	go func() {
		w.watch()
		failed <- struct{}{}
	}()
	timer := time.NewTimer(time.Millisecond * 500)
	select {
	case <-failed:
	case <-timer.C:
		t.Error("watchdog did not fail")
	}
}

func TestWatchingWorks(t *testing.T) {
	logger := testlogger.New(t)
	w, err := createTestWatchdog(true, logger)
	if err != nil {
		t.Fatal(err)
	}
	failed := make(chan struct{})
	go func() {
		w.watch()
		failed <- struct{}{}
	}()
	timer := time.NewTimer(time.Millisecond * 500)
	select {
	case <-failed:
		t.Error("watchdog failed")
	case <-timer.C:
	}
}
