package packets

import (
	"fmt"
	"net"
	"packets/internal/consts"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/ipv4"
)

type Peer struct {
	IP   net.IP
	Port int
}

func BroadcastAddr(ip net.IP, mask net.IPMask) net.IP {
	b := make(net.IP, len(ip))
	for i := range ip {
		b[i] = ip[i] | ^mask[i]
	}
	return b
}

func AskLAN(filename string) ([]Peer, error) {
	var peers []Peer
	query := []byte("Q:" + filename)

	pc, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return []Peer{}, err
	}
	defer pc.Close()

	if pconn := ipv4.NewPacketConn(pc); pconn != nil {
		_ = pconn.SetTTL(1)
	}

	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}

			bcast := BroadcastAddr(ipnet.IP.To4(), ipnet.Mask)
			dst := &net.UDPAddr{IP: bcast, Port: 1333}

			_, err = pc.WriteTo(query, dst)
			if err != nil {
				fmt.Printf(":: (%s) can't send to  %s: %s\n", ifc.Name, bcast, err.Error())
			}
		}
	}
	_ = pc.SetDeadline(time.Now().Add(consts.LANDeadline))
	buf := make([]byte, 1500)

	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			break
		}
		msg := string(buf[:n])

		if strings.HasPrefix(msg, "H:"+filename) {
			parts := strings.Split(msg, ":")
			port, _ := strconv.Atoi(parts[2])
			peers = append(peers, Peer{IP: addr.(*net.UDPAddr).IP, Port: port})
		}
	}
	return peers, nil
}
