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
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type probeResultType struct {
	err error
	ip  net.IP
}

func listToMap(list []net.IP) map[string]net.IP {
	ipMap := make(map[string]net.IP, len(list))
	for _, ip := range list {
		ipMap[ip.String()] = ip
	}
	return ipMap
}

func newLoadBalancer(config Config, backend RecordReadWriter,
	logger log.DebugLogger) (*LoadBalancer, error) {
	if config.FQDN == "" {
		return nil, errors.New("no FQDN specified")
	}
	if config.CheckInterval < time.Second*5 {
		config.CheckInterval = time.Second * 5
	}
	if config.TcpPort < 1 {
		return nil, errors.New("no TCP port number specified")
	}
	crandData := make([]byte, 8)
	if _, err := crand.Read(crandData); err != nil {
		return nil, err
	}
	seed, _ := binary.Varint(crandData)
	lb := &LoadBalancer{
		backend: backend,
		config:  config,
		logger:  logger,
		rand:    mrand.New(mrand.NewSource(seed)),
	}
	if myIP, err := util.GetMyIP(); err != nil {
		return nil, err
	} else {
		lb.myIP = myIP
		lb.myStringIP = myIP.String()
	}
	go lb.checkLoop()
	return lb, nil
}

func (lb *LoadBalancer) checkIP(ip net.IP) probeResultType {
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
func (lb *LoadBalancer) checkIPs(checkMap map[string]net.IP) map[string]net.IP {
	responseChannel := make(chan probeResultType, len(checkMap))
	for _, ip := range checkMap {
		go func(ipAddr net.IP) {
			responseChannel <- lb.checkIP(ipAddr)
		}(ip)
	}
	badMap := make(map[string]net.IP, len(checkMap))
	for range checkMap {
		response := <-responseChannel
		if response.err != nil {
			lb.logger.Printf("error probing: %s: %s\n",
				response.ip, response.err)
			badMap[response.ip.String()] = response.ip
		}
	}
	return badMap
}

func (lb *LoadBalancer) checkLoop() {
	for {
		if err := lb.check(); err != nil {
			lb.logger.Println(err)
		}
		// Sleep [0.75:1.25] * lb.checkInterval.
		time.Sleep((lb.config.CheckInterval>>2)*3 +
			(lb.config.CheckInterval>>9)*time.Duration(lb.rand.Int63n(256)))
	}
}

func (lb *LoadBalancer) check() error {
	checkList, err := lb.backend.ReadRecord(lb.config.FQDN)
	if err != nil {
		return err
	}
	startTime := time.Now()
	lb.logger.Debugf(1, "read DNS for: %s: %v\n", lb.config.FQDN, checkList)
	checkMap := listToMap(checkList)
	_, present := checkMap[lb.myStringIP]
	delete(checkMap, lb.myStringIP)
	badMap := lb.checkIPs(checkMap)
	if present &&
		len(badMap) < 1 &&
		time.Since(startTime) < lb.config.CheckInterval>>4 {
		lb.logger.Debugf(0, "no DNS changes for: %s (fast check)\n",
			lb.config.FQDN)
		return nil
	}
	oldList, err := lb.backend.ReadRecord(lb.config.FQDN)
	if err != nil {
		return err
	}
	oldMap := listToMap(oldList)
	newList := make([]net.IP, 0, len(oldList))
	foundMyself := false
	for _, ip := range oldList {
		ipString := ip.String()
		if ipString == lb.myStringIP {
			newList = append(newList, ip)
			foundMyself = true
		} else if _, ok := badMap[ipString]; !ok {
			newList = append(newList, ip)
		}
	}
	if !foundMyself {
		newList = append(newList, lb.myIP)
		lb.logger.Printf("adding my IP (%s) to DNS\n", lb.myStringIP)
	}
	noChanges := true
	if len(newList) != len(oldMap) {
		noChanges = false
	} else {
		for _, ip := range newList {
			if _, ok := oldMap[ip.String()]; !ok {
				noChanges = false
				break
			}
		}
	}
	if noChanges {
		lb.logger.Debugf(0, "no DNS changes for: %s\n", lb.config.FQDN)
		return nil
	}
	lb.logger.Printf("updating DNS for: %s: %v\n", lb.config.FQDN, newList)
	return lb.backend.WriteRecord(lb.config.FQDN, newList,
		lb.config.CheckInterval)
}
