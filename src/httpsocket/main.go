package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
)

type ConfigTOML struct {
	Config struct {
		DefaultHttpPort int    `toml:"defaultHttpPort"`
		DefaultCacheDir string `toml:"defaultCacheDir"`
	} `toml:"Config"`
}

func main() {
	var cfg ConfigTOML
	toml.Decode("opt/packets/packets/config.toml", &cfg)

	pid := os.Getpid()
	if err := os.WriteFile("./http.pid", []byte(fmt.Sprint(pid)), 0644); err != nil {
		fmt.Println("error saving subprocess pid", err)
	}

	fs := http.FileServer(http.Dir(cfg.Config.DefaultCacheDir))
	http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Config.DefaultHttpPort), nil))
}
