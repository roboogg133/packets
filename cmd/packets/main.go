package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"packets/configs"
	"packets/internal/consts"
	"packets/internal/utils"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

// init is doing some verifications
func init() {

	log.SetFlags(0)

	_, err := os.Stat(consts.DefaultLinux_d)
	if os.IsNotExist(err) {
		err := os.Mkdir(consts.DefaultLinux_d, 0755)
		if err != nil {
			if os.IsPermission(err) {
				log.Fatal("can't create packets root directory, please run as root")
			} else {
				log.Fatal(err)
			}
		}
	}

	_, err = os.Stat(filepath.Join(consts.DefaultLinux_d, "index.db"))
	if err != nil {

		if os.IsNotExist(err) {
			fmt.Println("index.db does not exist, try to use \"packets sync\"")
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
			db.Exec("CREATE TABLE IF NOT EXISTS packages (name TEXT NOT NULL UNIQUE PRIMARY KEY, version TEXT NOT NULL, dependencies TEXT NOT NULL DEFAULT '', family TEXT NOT NULL, serial INTEGER, package_d TEXT NOT NULL)")
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

var syncCmd = &cobra.Command{
	Use:   "sync [url]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Syncronizes with an remote index.db, and check if the data dir is changed",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getuid() != 0 {
			fmt.Println("please, run as root")
			return
		}
		syncUrl := consts.DefaultSyncUrl
		if len(args) > 0 {
			syncUrl = args[0]
		}

		DBB, err := utils.GetFileHTTP(syncUrl)
		if err != nil {
			log.Fatal(err)
		}

		if err := os.WriteFile(consts.IndexDB, DBB, 0774); err != nil {
			log.Fatal(err)
		}

		fmt.Println("Sucessifully sync!")
		os.Exit(0)
	},
}

type Quer1 struct {
	Name        string
	Version     string
	Description string
}

var installCmd = &cobra.Command{
	Use:   "install {package} [packages...]",
	Short: "Install a package",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := sql.Open("sqlite", consts.IndexDB)
		if err != nil {
			fmt.Println(err)
		}
		defer db.Close()

		for _, inputName := range args {

			var exist bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE name = ?)", inputName).Scan(&exist)
			if err != nil {
				if err != sql.ErrNoRows {
					log.Panic(err)
				}
			}
			if exist {

			}

			rows, err := db.Query("SELECT name, version, descriptionFROM packages WHERE query_name = ?", inputName)
			if err != nil {
				log.Fatal(err)

			}

			defer rows.Close()

			var pkgs []Quer1
			for rows.Next() {
				var q Quer1
				if err := rows.Scan(&q.Name, &q.Version, &q.Description); err != nil {
					log.Panic(err)
				}
				pkgs = append(pkgs, q)
			}
			switch len(pkgs) {
			case 0:
				log.Fatalf("can't find any results for %s\n", inputName)

			case 1:
				fmt.Printf(":: Founded 1 package for %s \n", inputName)

				fmt.Printf("Downloading %s \n", pkgs[0].Name)
				goto install

			default:

				fmt.Printf(":: Founded %d packages for (%s)\n Select 1 to install\n", len(pkgs), inputName)
				for i, q := range pkgs {
					fmt.Printf("[%d] %s : %s\n     %s\n", i, q.Name, q.Version, q.Description)
				}
				var choice int
			optionagain:
				fmt.Print(">> ")
				fmt.Fscan(bufio.NewReader(os.Stdin), &choice)
				if choice > len(pkgs) || choice < 0 {
					fmt.Println("invalid option")
					goto optionagain
				}

				return
			}

		install:
		}

	},
}

func main() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.Execute()
}
