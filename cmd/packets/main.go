package main

import (
	"bufio"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"packets/configs"
	"packets/internal/consts"
	"packets/internal/utils"
	packets "packets/pkg"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

// init is doing some verifications
func init() {
	log.SetPrefix("error: ")
	log.SetFlags(0)
	//log.SetFlags(log.Lshortfile)
	_, err := os.Stat(consts.DefaultLinux_d)
	if os.IsNotExist(err) {
		err := os.Mkdir(consts.DefaultLinux_d, 0777)
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
			if len(os.Args) > 1 && os.Args[0] != "sync" {
			} else {
				fmt.Println("index.db does not exist, try to use \"packets sync\"")
			}
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
			if _, err := db.Exec(consts.InstalledDatabaseSchema); err != nil {
				log.Fatal(err)
			}
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
		_, err := os.Stat(consts.IndexDB)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Fatal("index.db does not exist, try to use \"packets sync\"")
			}
		}
		f, err := os.OpenFile(consts.IndexDB, os.O_WRONLY, 0)
		if err != nil {
			log.Fatalf("can't open to write [ %s ]. Are you running packets as root?\n", consts.IndexDB)
		}
		f.Close()

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

		fmt.Printf(":: Sucessifully syncronized index.db with [ %s ]\n", syncUrl)
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
		_, err := os.Stat(consts.IndexDB)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Fatal("index.db does not exist, try to use \"packets sync\"")
			}
		}

		if os.Getuid() != 0 {
			log.Fatal("you must run this command as root")
		}

		db, err := sql.Open("sqlite", consts.IndexDB)
		if err != nil {
			fmt.Println(err)
		}
		defer db.Close()

		cfg, err := configs.GetConfigTOML()
		if err != nil {
			log.Fatal(err)
		}

		for _, inputName := range args {

			var exist bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE name = ?)", inputName).Scan(&exist)
			if err != nil {
				if err != sql.ErrNoRows {
					log.Panic(err)
				}
			}
			if exist {
				installed, err := utils.CheckIfPackageInstalled(inputName)
				if err != nil {
					log.Fatal(err)
				}
				if installed {
					log.Printf("package %s is already installed\n", inputName)
					continue
				}
				fmt.Printf("Checking dependencies of (%s)\n", inputName)
				dependenciesRaw, err := utils.GetDependencies(db, inputName)
				if err != nil {
					log.Fatal(err)
				}

				dependencies, err := utils.ResolvDependencies(dependenciesRaw)
				if err != nil {
					log.Fatal(err)
				}

				if len(dependencies) > 0 {
					var wg sync.WaitGroup
					var mu sync.Mutex
					fmt.Printf(":: Packets will install %s and %d dependencies\nPackages to install:\n", inputName, len(dependencies))
					fmt.Println(dependencies)
					fmt.Println("Are you sure? (y/N)")
					var a string
					fmt.Scanf("%s", &a)
					if a != "y" && a != "Y" {
						os.Exit(1)
					}
					for _, depn := range dependencies {
						wg.Add(1)
						go AyncFullInstall(depn, cfg.Config.StorePackages, filepath.Join(cfg.Config.Data_d, inputName), &wg, &mu)
					}

					wg.Wait()

				}
				fmt.Printf("Downloading (%s) \n", inputName)
				p, err := utils.GetPackage(inputName)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Printf(":: Installing (%s) \n", inputName)
				if err := packets.InstallPackage(p.PackageF, filepath.Join(cfg.Config.Data_d, inputName)); err != nil {
					log.Fatal(err)
				}

				if cfg.Config.StorePackages {
					_, err := p.Write()
					if err != nil {
						log.Fatal(err)
					}
					err = p.AddToInstalledDB(1, filepath.Join(cfg.Config.Data_d, inputName))
					if err != nil {
						log.Fatal(err)
					}
				} else {
					err := p.AddToInstalledDB(0, filepath.Join(cfg.Config.Data_d, inputName))
					if err != nil {
						log.Fatal(err)
					}
				}

				continue

			}

			rows, err := db.Query("SELECT id, version, description FROM packages WHERE query_name = ?", inputName)
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
				log.Fatalf("can't find any results for (%s)\n", inputName)

			case 1:
				fmt.Printf(":: Founded 1 package for %s \n", inputName)

				installed, err := utils.CheckIfPackageInstalled(pkgs[0].Name)
				if err != nil {
					log.Fatal(err)
				}

				if installed {
					log.Printf("Package %s is already installed\n", pkgs[0].Name)
					continue
				}

				fmt.Printf("Checking dependencies of (%s)\n", pkgs[0].Name)
				dependenciesRaw, err := utils.GetDependencies(db, inputName)
				if err != nil {
					log.Fatal(err)
				}

				dependencies, err := utils.ResolvDependencies(dependenciesRaw)
				if err != nil {
					log.Fatal(err)
				}

				if len(dependencies) > 0 {
					var wg sync.WaitGroup
					var mu sync.Mutex

					fmt.Printf(":: Packets will install %s and %d dependencies\nPackages to install:\n", pkgs[0].Name, len(dependencies))
					fmt.Println(dependencies)
					fmt.Println("Are you sure? (y/N)")
					var a string
					fmt.Scanf("%s", &a)
					if a != "y" && a != "Y" {
						os.Exit(1)
					}
					for _, depn := range dependencies {
						wg.Add(1)
						go AyncFullInstall(depn, cfg.Config.StorePackages, filepath.Join(cfg.Config.Data_d, inputName), &wg, &mu)
					}

					wg.Wait()

				}

				fmt.Printf("Downloading %s \n", pkgs[0].Name)
				p, err := utils.GetPackage(pkgs[0].Name)
				if err != nil {
					log.Fatal(err)
				}

				cfg, err := configs.GetConfigTOML()
				if err != nil {
					log.Fatal(err)
				}

				fmt.Printf(":: Installing %s \n", pkgs[0].Name)
				if err := packets.InstallPackage(p.PackageF, filepath.Join(cfg.Config.Data_d, pkgs[0].Name)); err != nil {
					log.Fatal(err)
				}

				if cfg.Config.StorePackages {
					_, err := p.Write()
					if err != nil {
						log.Fatal(err)
					}
					err = p.AddToInstalledDB(1, filepath.Join(cfg.Config.Data_d, pkgs[0].Name))
					if err != nil {
						log.Fatal(err)
					}
				} else {
					err := p.AddToInstalledDB(0, filepath.Join(cfg.Config.Data_d, pkgs[0].Name))
					if err != nil {
						log.Fatal(err)
					}
				}
				continue

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

				installed, err := utils.CheckIfPackageInstalled(pkgs[choice].Name)
				if err != nil {
					log.Fatal(err)
				}
				if installed {
					log.Printf("package %s is already installed\n", pkgs[choice].Name)
					continue
				}

				fmt.Printf("Checking dependencies of (%s)\n", pkgs[choice].Name)
				dependenciesRaw, err := utils.GetDependencies(db, inputName)
				if err != nil {
					log.Fatal(err)
				}

				dependencies, err := utils.ResolvDependencies(dependenciesRaw)
				if err != nil {
					log.Fatal(err)
				}

				if len(dependencies) > 0 {
					var wg sync.WaitGroup
					var mu sync.Mutex

					fmt.Printf(":: Packets will install %s and %d dependencies\nPackages to install:\n", pkgs[choice].Name, len(dependencies))
					fmt.Println(dependencies)
					fmt.Println("Are you sure? (y/N)")
					var a string
					fmt.Scanf("%s", &a)
					if a != "y" && a != "Y" {
						os.Exit(1)
					}
					for _, depn := range dependencies {
						wg.Add(1)
						go AyncFullInstall(depn, cfg.Config.StorePackages, filepath.Join(cfg.Config.Data_d, pkgs[choice].Name), &wg, &mu)
					}

					wg.Wait()

				}

				fmt.Printf("Downloading %s \n", pkgs[choice].Name)
				p, err := utils.GetPackage(pkgs[choice].Name)
				if err != nil {
					log.Fatal(err)
				}

				cfg, err := configs.GetConfigTOML()
				if err != nil {
					log.Fatal(err)
				}

				fmt.Printf(":: Installing (%s) \n", pkgs[choice].Name)
				if err := packets.InstallPackage(p.PackageF, filepath.Join(cfg.Config.Data_d, pkgs[choice].Name)); err != nil {
					log.Fatal(err)
				}

				if cfg.Config.StorePackages {
					_, err := p.Write()
					if err != nil {
						log.Fatal(err)
					}
					err = p.AddToInstalledDB(1, filepath.Join(cfg.Config.Data_d, pkgs[choice].Name))
					if err != nil {
						log.Fatal(err)
					}
				} else {
					err := p.AddToInstalledDB(0, filepath.Join(cfg.Config.Data_d, pkgs[choice].Name))
					if err != nil {
						log.Fatal(err)
					}
				}
				continue
			}
		}
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove {package name}[package name...] ",
	Args:  cobra.MinimumNArgs(1),
	Short: "Remove a package from the given names",
	Run: func(cmd *cobra.Command, args []string) {

		if os.Getuid() != 0 {
			log.Fatal("you must run this command as root")
		}

		fmt.Print("WARNING: This command will remove permanently this packages, are you sure? (y/N) ")
		var a string
		fmt.Scanf("%s", &a)
		if a != "y" && a != "Y" {
			os.Exit(1)
		}

		for _, pkgName := range args {

			installed, err := utils.CheckIfPackageInstalled(pkgName)
			if err != nil {
				log.Fatal(err)
			}
			if installed {
				db, err := sql.Open("sqlite", consts.InstalledDB)
				if err != nil {
					log.Fatal(err)
				}
				var packageDir string
				if err := db.QueryRow("SELECT package_d FROM packages WHERE query_name = ? OR id = ?", pkgName, pkgName).Scan(&packageDir); err != nil {
					log.Fatal(err)
				}

				f, err := os.Open(filepath.Join(packageDir, "manifest.toml"))
				if err != nil {
					log.Fatal(err)
				}

				manifest, err := utils.ManifestFileRead(f)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(":: Removing", pkgName)

				packets.ExecuteRemoveScript(filepath.Join(packageDir, manifest.Hooks.Remove))

				if err := os.RemoveAll(packageDir); err != nil {
					log.Fatal(err)
				}

				if err := utils.RemoveFromInstalledDB(pkgName); err != nil {
					log.Fatal(err)
				}

				fmt.Println("Sucessifully removed")

				os.Exit(0)
			}
			log.Fatalf("%s not installed", pkgName)
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: "List all installed packages",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := sql.Open("sqlite", consts.InstalledDB)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM packages").Scan(&count); err != nil {
			log.Fatal(err)
		}

		rows, err := db.Query("SELECT query_name, id, version, description, package_d, os, arch FROM packages")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		fmt.Printf(":: Listing all %d packages installed:\n\n", count)

		for rows.Next() {
			var queryName, name, version, description, packageDir, os, arch string
			if err := rows.Scan(&queryName, &name, &version, &description, &packageDir, &os, &arch); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("  Package %s \n   ├──Id: %s\n   ├──Version: %s \n   ├──Package dir: %s\n   ├──OS: %s\n   ├──Arch: %s\n   └──Description: %s\n", queryName, name, version, packageDir, os, arch, description)
		}
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Search for packages in the index.db",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := sql.Open("sqlite", consts.IndexDB)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM packages").Scan(&count); err != nil {
			log.Fatal(err)
		}

		var rows *sql.Rows

		if len(args) > 0 {
			rows, err = db.Query("SELECT query_name, id, version, description, os, arch FROM packages WHERE name LIKE ? OR description LIKE ? OR query_name LIKE ?", args[0], args[0], args[0])
			if err != nil {
				log.Fatal(err)
			}
			defer rows.Close()
		} else {

			rows, err = db.Query("SELECT query_name, id, version, description, os, arch FROM packages")
			if err != nil {
				log.Fatal(err)
			}
			defer rows.Close()
		}

		fmt.Printf(":: Listing all %d packages:\n\n", count)

		for rows.Next() {
			var queryName, name, version, description, os, arch string
			if err := rows.Scan(&queryName, &name, &version, &description, &os, &arch); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("  Package %s \n   ├──Query name: %s\n   ├──Version: %s \n   ├──OS: %s\n   ├──Arch: %s\n   └──Description: %s\n", name, queryName, version, os, arch, description)
		}
	},
}

func main() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.Execute()
}

func AyncFullInstall(dep string, storePackages bool, installPath string, wg *sync.WaitGroup, mu *sync.Mutex) {
	defer wg.Done()

	fmt.Printf(" downloading %s \n", dep)
	p, err := utils.GetPackage(dep)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(" installing %s \n", dep)

	if err := packets.InstallPackage(p.PackageF, installPath); err != nil {
		log.Fatal(err)
	}
	if storePackages {
		_, err := p.Write()
		if err != nil {
			log.Fatal(err)
			return
		}
		mu.Lock()
		defer mu.Unlock()

		err = p.AddToInstalledDB(1, installPath)
		if err != nil {
			log.Fatal(err)
			return
		}
	} else {

		mu.Lock()
		defer mu.Unlock()

		err := p.AddToInstalledDB(0, installPath)
		if err != nil {
			log.Println(err)
			return
		}
	}
}
