package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"packets/internal"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type ConfigTOML struct {
	Config struct {
		DefaultHttpPort int    `toml:"httpPort"`
		DefaultCacheDir string `toml:"cacheDir"`
	} `toml:"Config"`
}

var cfg ConfigTOML

func CheckDownloaded(filename string) bool {

	_, err := os.Stat(filepath.Join(cfg.Config.DefaultCacheDir))
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}

}

func main() {
	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(internal.PacketsPackageDir(), "udp.pid"), []byte(fmt.Sprint(pid)), 0644); err != nil {
		fmt.Println("error saving subprocess pid", err)
	}
	toml.Decode(filepath.Join(internal.PacketsPackageDir(), "config.toml"), &cfg)

	addr := net.UDPAddr{IP: net.IPv4zero, Port: 1333}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	buf := make([]byte, 1500)

	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("error creating udp socket", err)
		}
		msg := string(buf[:n])
		if !strings.HasPrefix(msg, "Q:") {
			continue
		}
		filename := strings.TrimPrefix(msg, "Q:")
		if CheckDownloaded(filename) {
			reply := fmt.Sprintf("H:%s:%d", filename, 9123)
			conn.WriteToUDP([]byte(reply), remote)
		}
	}
}
