package dnslb

import (
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

type blockedType struct {
	IP           string
	IpExpires    time.Time
	OwnerId      string
	OwnerExpires time.Time
}

func parseBlocked(txts []string) (*blockedType, error) {
	if len(txts) < 1 {
		return nil, nil
	} else if len(txts) < 2 {
		return nil, errors.New("wrong number of values")
	}
	var blocked blockedType
	for _, txt := range txts {
		txt = strings.TrimSpace(txt)
		splitTxt := strings.Split(txt, "=")
		if len(splitTxt) != 2 {
			return nil, fmt.Errorf("bad split for: %s", txt)
		}
		splitTxt[0] = strings.TrimSpace(splitTxt[0])
		splitTxt[1] = strings.TrimSpace(splitTxt[1])
		switch splitTxt[0] {
		case "IP":
			blocked.IP = splitTxt[1]
		case "IpExpires":
			expires, err := time.Parse(time.RFC3339, splitTxt[1])
			if err != nil {
				return nil, err
			}
			blocked.IpExpires = expires
		case "OwnerId":
			blocked.OwnerId = splitTxt[1]
		case "OwnerExpires":
			expires, err := time.Parse(time.RFC3339, splitTxt[1])
			if err != nil {
				return nil, err
			}
			blocked.OwnerExpires = expires
		}
	}
	if blocked.OwnerId == "" {
		return nil, errors.New("no OwnerId specified")
	}
	if blocked.OwnerExpires.IsZero() {
		return nil, errors.New("no owner expiration time specified")
	}
	if time.Until(blocked.OwnerExpires) <= 0 {
		return nil, errors.New("expired owner")
	}
	return &blocked, nil
}

func rollingReplace(config Config, params Params, region string,
	logger log.DebugLogger) error {
	lb := &LoadBalancer{
		config: config,
		p:      params,
	}
	regionalIPs, ttl, err := lb.getRegionalIPs()
	if err != nil {
		return err
	}
	if lb.config.CheckInterval < time.Second {
		lb.config.CheckInterval = ttl
	}
	regionalIpList := make([]string, 0, len(regionalIPs))
	anyBlocked := false
	for ip := range regionalIPs {
		blocked, err := lb.checkBlocked(ip)
		if err != nil {
			return err
		}
		if blocked > 0 {
			anyBlocked = true
			logger.Printf("%s is blocked\n", ip)
		}
		regionalIpList = append(regionalIpList, ip)
	}
	if anyBlocked {
		return errors.New(
			"some IP(s) are blocked: another rolling replace is active")
	}
	logger.Debugf(0, "%s: regional IPs: %v\n", config.FQDN, regionalIpList)
	if len(regionalIpList) < 2 {
		return fmt.Errorf("need 2+ regional IPs, have: %v\n", regionalIpList)
	}
	crandData := make([]byte, 4)
	if _, err := crand.Read(crandData); err != nil {
		return err
	}
	myId := hex.EncodeToString(crandData)
	for _, ip := range regionalIpList {
		if err := lb.replaceOne(myId, ip, ttl, len(regionalIPs)); err != nil {
			return err
		}
	}
	if err := lb.cleanupBlock(); err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) block(myId, ip string, ttl time.Duration) error {
	fqdn := lb.generateBlockedFqdn()
	blocked, err := lb.getBlockedData(fqdn)
	if err != nil {
		return err
	}
	if blocked != nil && blocked.OwnerId != myId {
		return fmt.Errorf("blocked by another owner: %s", blocked.OwnerId)
	}
	var txts []string
	interval := ttl * 2
	if ip != "" {
		txts = append(txts,
			"IP="+ip,
			"IpExpires="+time.Now().Add(interval).Format(time.RFC3339))
	}
	txts = append(txts,
		"OwnerId="+myId,
		"OwnerExpires="+time.Now().Add(ttl*5).Format(time.RFC3339))
	rrw := lb.p.RecordReadWriter
	if err := rrw.WriteRecords(fqdn, "TXT", txts, ttl); err != nil {
		return fmt.Errorf("error writing: %s: TXT=%v", fqdn, txts)
	}
	if ip == "" {
		lb.p.Logger.Printf("locked for: %s\n", ttl*5)
	} else {
		lb.p.Logger.Printf("blocked: %s for: %s\n", ip, interval)
	}
	return nil
}

// Returns duration blocked, else <= 0.
func (lb *LoadBalancer) checkBlocked(ip string) (time.Duration, error) {
	fqdn := lb.generateBlockedFqdn()
	blocked, err := lb.getBlockedData(fqdn)
	if err != nil {
		return 0, err
	}
	if blocked == nil {
		return 0, nil
	}
	if ip != blocked.IP {
		return 0, nil
	}
	if blocked.IpExpires.IsZero() {
		return 0, nil
	}
	return time.Until(blocked.IpExpires), nil
}

func (lb *LoadBalancer) cleanupBlock() error {
	fqdn := lb.generateBlockedFqdn()
	rrw := lb.p.RecordReadWriter
	if err := rrw.DeleteRecords(fqdn, "TXT"); err != nil {
		return err
	}
	lb.p.Logger.Printf("cleaned up: %s\n", fqdn)
	return nil
}

func (lb *LoadBalancer) generateBlockedFqdn() string {
	return "_blocked." + lb.config.FQDN
}

func (lb *LoadBalancer) getBlockedData(fqdn string) (*blockedType, error) {
	rrw := lb.p.RecordReadWriter
	txts, _, err := rrw.ReadRecords(fqdn, "TXT")
	if err != nil {
		return nil, err
	}
	blocked, err := parseBlocked(txts)
	if err != nil {
		if err := rrw.DeleteRecords(fqdn, "TXT"); err != nil {
			return nil, err
		}
		lb.p.Logger.Printf("deleted: %s: %s\n", fqdn, err)
		return nil, nil
	}
	return blocked, nil
}

func (lb *LoadBalancer) getRegionalIPs() (
	map[string]struct{}, time.Duration, error) {
	ipList, ttl, err := lb.p.RecordReadWriter.ReadRecords(lb.config.FQDN, "A")
	if err != nil {
		return nil, 0, err
	}
	ips := make(map[string]struct{}, len(ipList))
	for _, ip := range ipList {
		ips[ip] = struct{}{}
	}
	regionalIPs, err := lb.p.RegionFilter.Filter(ips)
	if err != nil {
		return nil, 0, err
	}
	return regionalIPs, ttl, nil
}

func (lb *LoadBalancer) replaceOne(myId, ip string, ttl time.Duration,
	numRequired int) error {
	newTtl := time.Second * 5
	if newTtl > ttl {
		newTtl = ttl
	}
	// Grab lock and block the instance from adding itself to DNS.
	if err := lb.block(myId, ip, ttl); err != nil {
		return err
	}
	// Remove instance from DNS.
	oldList, _, err := lb.p.RecordReadWriter.ReadRecords(lb.config.FQDN, "A")
	if err != nil {
		return err
	}
	newList := make([]string, 0, len(oldList)-1)
	for _, oldIP := range oldList {
		if oldIP != ip {
			newList = append(newList, oldIP)
		}
	}
	err = lb.p.RecordReadWriter.WriteRecords(lb.config.FQDN, "A", newList,
		newTtl)
	if err != nil {
		return err
	}
	lb.p.Logger.Printf("removed: %s from: %s\n", ip, lb.config.FQDN)
	// Wait for TTL to expire.
	lb.p.Logger.Printf("sleeping for: %s before destroying: %s\n", ttl, ip)
	time.Sleep(ttl)
	// Destroy instance which should no longer be visable via DNS.
	ipMap := map[string]struct{}{ip: struct{}{}}
	if err := lb.p.Destroyer.Destroy(ipMap); err != nil {
		return err
	}
	lb.p.Logger.Printf("destroyed: %s, now waiting for replacement\n", ip)
	// Wait for required number of healthy instances, keeping the lock fresh.
	for {
		time.Sleep(ttl >> 2)
		if err := lb.block(myId, "", ttl); err != nil {
			return err
		}
		ips, _, err := lb.getRegionalIPs()
		if err != nil {
			return err
		}
		if len(ips) < numRequired {
			lb.p.Logger.Printf("only %d instances registered, need %d\n",
				len(ips), numRequired)
			continue
		}
		badIPs := lb.checkIPs(ips)
		if len(badIPs) < 1 {
			break
		}
		lb.p.Logger.Printf("unhealthy instances: %v\n", badIPs)
	}
	return nil
}
