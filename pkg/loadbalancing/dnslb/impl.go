package dnslb

import (
	crand "crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	mrand "math/rand"
	"net"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/net/util"
)

type probeResultType struct {
	err error
	ip  string
}

func listToMap(list []string) map[string]struct{} {
	ipMap := make(map[string]struct{}, len(list))
	for _, ip := range list {
		ipMap[ip] = struct{}{}
	}
	return ipMap
}

func newLoadBalancer(config Config, params Params) (*LoadBalancer, error) {
	if config.FQDN == "" {
		return nil, errors.New("no FQDN specified")
	}
	if config.CheckInterval < time.Second*5 {
		config.CheckInterval = time.Second * 5
	}
	if config.MaximumFailures < 1 {
		config.MaximumFailures = 60
	}
	if config.MinimumFailures < 1 {
		config.MinimumFailures = 3
	}
	if config.TcpPort < 1 {
		return nil, errors.New("no TCP port number specified")
	}
	if params.Destroyer == nil {
		params.Destroyer = nullInterface
	}
	if params.RecordReadWriter == nil {
		return nil, errors.New("no RecordReadWriter specified")
	}
	if params.RegionFilter == nil {
		params.RegionFilter = nullInterface
	}
	crandData := make([]byte, 8)
	if _, err := crand.Read(crandData); err != nil {
		return nil, err
	}
	seed, _ := binary.Varint(crandData)
	lb := &LoadBalancer{
		config:   config,
		failures: make(map[string]uint),
		p:        params,
		rand:     mrand.New(mrand.NewSource(seed)),
	}
	if myIP, err := util.GetMyIP(); err != nil {
		return nil, err
	} else {
		lb.myIP = myIP.String()
	}
	go lb.checkLoop()
	return lb, nil
}

func (lb *LoadBalancer) checkIP(ip string) probeResultType {
	addr := fmt.Sprintf("%s:%d", ip, lb.config.TcpPort)
	conn, err := net.DialTimeout("tcp", addr, lb.config.CheckInterval>>2)
	if err != nil {
		return probeResultType{err: err, ip: ip}
	}
	defer conn.Close()
	if lb.config.DoTLS {
		tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
		defer tlsConn.Close()
		if err := tlsConn.Handshake(); err != nil {
			return probeResultType{err: err, ip: ip}
		}
	}
	return probeResultType{err: nil, ip: ip}
}

// Probe each IP, return bad IPs.
func (lb *LoadBalancer) checkIPs(
	checkMap map[string]struct{}) map[string]struct{} {
	responseChannel := make(chan probeResultType, len(checkMap))
	for ip := range checkMap {
		go func(ipAddr string) {
			responseChannel <- lb.checkIP(ipAddr)
		}(ip)
	}
	badMap := make(map[string]struct{}, len(checkMap))
	for range checkMap {
		response := <-responseChannel
		if response.err != nil {
			lb.p.Logger.Printf("error probing: %s: %s\n",
				response.ip, response.err)
			badMap[response.ip] = struct{}{}
		}
	}
	return badMap
}

func (lb *LoadBalancer) checkLoop() {
	for {
		if err := lb.check(); err != nil {
			lb.p.Logger.Println(err)
		}
		// Sleep [0.75:1.25] * lb.checkInterval.
		time.Sleep((lb.config.CheckInterval>>2)*3 +
			(lb.config.CheckInterval>>9)*time.Duration(lb.rand.Int63n(256)))
	}
}

func (lb *LoadBalancer) check() error {
	checkList, err := lb.p.RecordReadWriter.ReadRecord(lb.config.FQDN)
	if err != nil {
		return err
	}
	startTime := time.Now()
	lb.p.Logger.Debugf(1, "read DNS for: %s: %v\n", lb.config.FQDN, checkList)
	checkMap := listToMap(checkList)
	_, present := checkMap[lb.myIP]
	delete(checkMap, lb.myIP)
	badMap := lb.checkIPs(checkMap)
	for ip := range lb.failures { // Clean up old failures.
		if _, ok := badMap[ip]; !ok {
			delete(lb.failures, ip)
		}
	}
	for ip := range badMap {
		if lb.failures[ip]++; lb.failures[ip] < lb.config.MinimumFailures {
			delete(badMap, ip) // Has not been bad long enough.
		}
	}
	if present &&
		len(badMap) < 1 &&
		time.Since(startTime) < lb.config.CheckInterval>>4 {
		lb.p.Logger.Debugf(0, "no DNS changes for: %s (fast check)\n",
			lb.config.FQDN)
		return nil
	}
	removeMap, err := lb.p.RegionFilter.Filter(badMap)
	if err != nil {
		return err
	}
	if lb.config.MaximumFailures > lb.config.MinimumFailures {
		for ip := range badMap {
			if lb.failures[ip] > lb.config.MaximumFailures {
				removeMap[ip] = struct{}{}
			}
		}
	}
	if err := lb.p.Destroyer.Destroy(removeMap); err != nil {
		return err
	}
	oldList, err := lb.p.RecordReadWriter.ReadRecord(lb.config.FQDN)
	if err != nil {
		return err
	}
	oldMap := listToMap(oldList)
	newList := make([]string, 0, len(oldList))
	foundMyself := false
	for _, ip := range oldList {
		if ip == lb.myIP {
			newList = append(newList, ip)
			foundMyself = true
		} else if _, ok := removeMap[ip]; !ok {
			newList = append(newList, ip)
		} else {
			delete(lb.failures, ip) // Reset failure count.
		}
	}
	if !foundMyself {
		newList = append(newList, lb.myIP)
		lb.p.Logger.Printf("adding my IP (%s) to DNS\n", lb.myIP)
	}
	noChanges := true
	if len(newList) != len(oldMap) {
		noChanges = false
	} else {
		for _, ip := range newList {
			if _, ok := oldMap[ip]; !ok {
				noChanges = false
				break
			}
		}
	}
	if noChanges {
		lb.p.Logger.Debugf(0, "no DNS changes for: %s\n", lb.config.FQDN)
		return nil
	}
	lb.p.Logger.Printf("updating DNS for: %s: %v\n", lb.config.FQDN, newList)
	return lb.p.RecordReadWriter.WriteRecord(lb.config.FQDN, newList,
		lb.config.CheckInterval)
}
