package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"packets/configs"
	"packets/internal/consts"
	"path/filepath"
	"strings"
)

func CheckDownloaded(filename string) bool {

	cfg, err := configs.GetConfigTOML()
	if err != nil {
		log.Fatal(err)
	}

	_, err = os.Stat(filepath.Join(cfg.Config.Cache_d, filename))
	return err == nil
}

func main() {
	log.Println("Program started")
	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(consts.DefaultLinux_d, "udp.pid"), []byte(fmt.Sprint(pid)), 0664); err != nil {
		fmt.Println("error saving subprocess pid", err)
	}
	cfg, err := configs.GetConfigTOML()
	if err != nil {
		log.Fatal(err)
	}

	addr := net.UDPAddr{IP: net.IPv4zero, Port: 1333}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Started connection listener")
	defer conn.Close()
	buf := make([]byte, 1500)

	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("error creating udp socket", err)
		}
		msg := string(buf[:n])
		log.Printf("Received message : %s\n", msg)
		if !strings.HasPrefix(msg, "Q:") {
			log.Println("error: invalid message, this message don't follow the protocol")
			continue
		}
		filename := strings.TrimPrefix(msg, "Q:")
		if CheckDownloaded(filename) {
			reply := fmt.Sprintf("H:%s:%d", filename, cfg.Config.HttpPort)
			log.Printf("Package founded in cache dir, answering with: '%s'\n", reply)
			conn.WriteToUDP([]byte(reply), remote)
		}
	}
}
