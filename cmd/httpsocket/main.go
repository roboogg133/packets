package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"packets/configs"
	"packets/internal/consts"
	"path/filepath"
)

type ConfigTOML struct {
	Config struct {
		DefaultHttpPort int    `toml:"httpPort"`
		DefaultCacheDir string `toml:"cacheDir"`
	} `toml:"Config"`
}

func main() {

	cfg, err := configs.GetConfigTOML()
	if err != nil {
		log.Fatal(err)
	}

	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(consts.DefaultLinux_d, "http.pid"), []byte(fmt.Sprint(pid)), 0664); err != nil {
		fmt.Println("error saving subprocess pid", err)
	}

	fs := http.FileServer(http.Dir(cfg.Config.Cache_d))
	http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Config.HttpPort), nil))
}
