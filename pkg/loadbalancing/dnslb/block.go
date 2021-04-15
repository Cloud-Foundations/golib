package dnslb

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

func block(config Config, params Params, ip string, duration time.Duration,
	cancelChannel <-chan struct{}, logger log.DebugLogger) error {
	if duration < time.Minute {
		return fmt.Errorf("duration: %s is under a minute", duration)
	} else if duration > time.Hour {
		return fmt.Errorf("duration: %s is over an hour", duration)
	}
	lb := &LoadBalancer{
		config: config,
		p:      params,
	}
	crandData := make([]byte, 4)
	if _, err := crand.Read(crandData); err != nil {
		return err
	}
	myId := hex.EncodeToString(crandData)
	stopTime := time.Now().Add(duration)
	for keepGoing := true; keepGoing; {
		if time.Until(stopTime) <= 0 {
			break
		}
		if err := lb.block(myId, ip, time.Minute); err != nil {
			return err
		}
		timer := time.NewTimer(time.Minute)
		select {
		case <-cancelChannel:
			keepGoing = false
			timer.Stop()
		case <-timer.C:
		}
	}
	return lb.cleanupBlock()
}
