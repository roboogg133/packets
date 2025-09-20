package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"packets/configs"
	"packets/internal/consts"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

// init is doing some verifications
func init() {

	_, err := os.Stat(consts.DefaultLinux_d)
	if os.IsNotExist(err) {
		err := os.Mkdir(consts.DefaultLinux_d, 0644)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Println("can't create packets root directory, please run as root")
				os.Exit(1)
			} else {
				log.Fatal(err)
			}
		}
	}

	_, err = os.Stat(filepath.Join(consts.DefaultLinux_d, "index.db"))
	if err != nil {

		if os.IsNotExist(err) {
			fmt.Println("index.db does not exist use \"packets sync\"")
		} else {
			log.Fatal(err)
		}
	}

	_, err = os.Stat(filepath.Join(consts.DefaultLinux_d, "installed.db"))
	if err != nil {
		if os.IsNotExist(err) {
			db, err := sql.Open("sqlite", filepath.Join(consts.DefaultLinux_d, "installed.db"))
			if err != nil {
				log.Fatal(db)
			}
			defer db.Close()
			db.Exec("CREATE TABLE IF NOT EXISTS packages (realname TEXT NOT NULL UNIQUE PRIMARY KEY, version TEXT NOT NULL, dependencies TEXT, name TEXT, family TEXT NOT NULL, serial INTEGER, package_d TEXT NOT NULL)")
		} else {
			log.Fatal(err)
		}
	}

	_, err = os.Stat(filepath.Join(consts.DefaultLinux_d, "config.toml"))
	if os.IsNotExist(err) {
		f, err := os.Create(filepath.Join(consts.DefaultLinux_d, "config.toml"))
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()

		encoder := toml.NewEncoder(f)

		cfg, err := configs.DefaultConfigTOML()
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
var installCmd = &cobra.Command{
	Use:   "install {package} [packages...]",
	Short: "Install a package",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func main() {
	rootCmd.AddCommand(installCmd)
	rootCmd.Execute()
}
