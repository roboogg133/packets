package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"packets/internal"

	"github.com/BurntSushi/toml"
)

type ConfigTOML struct {
	Config struct {
		DefaultHttpPort int    `toml:"httpPort"`
		DefaultCacheDir string `toml:"cacheDir"`
	} `toml:"Config"`
}

func main() {

	internal.PacketsPackageDir()
	var cfg ConfigTOML
	toml.Decode(filepath.Join(internal.PacketsPackageDir(), "config.toml"), &cfg)

	pid := os.Getpid()
	if err := os.WriteFile(filepath.Join(internal.PacketsPackageDir(), "http.pid"), []byte(fmt.Sprint(pid)), 0644); err != nil {
		fmt.Println("error saving subprocess pid", err)
	}

	fs := http.FileServer(http.Dir(cfg.Config.DefaultCacheDir))
	http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Config.DefaultHttpPort), nil))
}
