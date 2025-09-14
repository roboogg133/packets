package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"packets/internal"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

// Consts

const DefaultLinux_d = "/etc/packets"
const DefaultLANDeadline = 2 * time.Second

// Errors

var ErrResponseNot200OK = errors.New("the request is not 200, download failed")

// Types

type ConfigTOML struct {
	Config struct {
		HttpPort int    `toml:"httpPort"`
		Cache_d  string `toml:"cache_d"`
		Data_d   string `toml:"data_d"`
		Bin_d    string `toml:"bin_d"`
	} `toml:"Config"`
}

// init is doing some verifications
func init() {

	_, err := os.Stat(DefaultLinux_d)
	if os.IsNotExist(err) {
		err := os.Mkdir(DefaultLinux_d, 0644)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Println("can't create packets root directory, please run as root")
				os.Exit(1)
			} else {
				log.Fatal(err)
			}
		}
	}

	_, err = os.Stat(filepath.Join(DefaultLinux_d, "index.db"))
	if err != nil {

		if os.IsNotExist(err) {
			fmt.Println("index.db does not exist use \"packets sync\"")
		} else {
			log.Fatal(err)
		}
	}

	_, err = os.Stat(filepath.Join(DefaultLinux_d, "installed.db"))
	if err != nil {
		if os.IsNotExist(err) {
			db, err := sql.Open("sqlite", filepath.Join(DefaultLinux_d, "installed.db"))
			if err != nil {
				log.Fatal(db)
			}
			defer db.Close()
			db.Exec("CREATE TABLE IF NOT EXISTS packages (realname TEXT NOT NULL UNIQUE PRIMARY KEY, version TEXT NOT NULL, dependencies TEXT, name TEXT, family TEXT NOT NULL, serial INTEGER, package_d TEXT NOT NULL)")
		} else {
			log.Fatal(err)
		}
	}

	_, err = os.Stat(filepath.Join(DefaultLinux_d, "config.toml"))
	if os.IsNotExist(err) {
		f, err := os.Create(filepath.Join(DefaultLinux_d, "config.toml"))
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()

		encoder := toml.NewEncoder(f)

		cfg, err := internal.DefaultConfigTOML()
		if err != nil {
			log.Fatal(err)
		}

		if err = encoder.Encode(*cfg); err != nil {
			log.Fatal(err)
		}
	}
}

// COBRA CMDS

var rootCmd = &cobra.Command{Use: "packets"}
var installCmd = &cobra.Command{}
