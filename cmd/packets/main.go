package main

import (
	"archive/tar"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	_ "github.com/mattn/go-sqlite3"
	"github.com/roboogg133/packets/cmd/packets/database"
	"github.com/roboogg133/packets/cmd/packets/decompress"
	"github.com/roboogg133/packets/cmd/packets/lockfile"
	"github.com/roboogg133/packets/cmd/packets/repo"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
	"github.com/spf13/cobra"
)

var verbosityLevel string

func GrantPrivilegies() {
	if os.Geteuid() != 0 {
		fmt.Println("error: this operation must be run as root")
		os.Exit(1)
	}
}

var Config *PacketsConfiguration

var rootCmd = &cobra.Command{
	Use:   "packets",
	Short: "A tool for managing packets",
	Long:  "A multiplatform package manager",
}

var executeCmd = &cobra.Command{
	Use:   "execute {path}",
	Short: "Installs a package from a given .pkt file",
	Long:  "Installs a package from a given .pkt file",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		GrantPrivilegies()
		return GetConfiguration()
	},
	Run: func(cmd *cobra.Command, args []string) {
		for _, v := range args {
			var pkg packet.PacketLua

			if !strings.HasSuffix(v, ".pkt") {
				fmt.Printf("error: %s is not a valid Packets packet file\n", v)
				os.Exit(1)
			}

			contentBlob, err := os.Open(v)
			if err != nil {
				fmt.Printf("error: %s could not be read\n", v)
				os.Exit(1)
			}
			defer contentBlob.Close()
			pkg, err = packet.ReadPacketFromZSTDF(contentBlob, &packet.Config{BinDir: Config.BinDir})
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			rootdir, err := filepath.Abs(filepath.Join(PackageRootDir, pkg.Name+"@"+pkg.Version))
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}
			sourcesdir, err := filepath.Abs(filepath.Join(rootdir, "src"))
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}
			configs := &packet.Config{
				BinDir:     Config.BinDir,
				RootDir:    rootdir,
				SourcesDir: sourcesdir,
			}

			db, err := sql.Open("sqlite3", InternalDB)
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}
			defer db.Close()

			database.PrepareDataBase(db)

			if installed, err := database.SearchIfIsInstalled(pkg.Name, db); err == nil {
				if installed {
					fmt.Printf("=> package %s is already installed\n", pkg.Name)
					continue
				}
			} else {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			backupDir, err := filepath.Abs(".")
			_ = ChangeToNoPermission()
			_ = os.MkdirAll(configs.RootDir, 0755)
			contentBlob.Seek(0, io.SeekStart)
			if err := decompress.Decompress(contentBlob, configs.RootDir, filepath.Base(v)); err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}
			_ = os.MkdirAll(configs.SourcesDir, 0755)

			if err := DownloadSource(&pkg.GlobalSources, configs, nil); err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			if pkg.Plataforms != nil {
				if plataform, exists := pkg.Plataforms[packet.OperationalSystem(runtime.GOOS)]; exists {
					if err := DownloadSource(&plataform.Sources, configs, nil); err != nil {
						fmt.Printf("error: %s", err.Error())
						os.Exit(1)
					}
				}
			}
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			pkg.ExecuteBuild(configs)
			pkg.ExecuteInstall(configs)
			_ = ElevatePermission()

			os.Chdir(backupDir)

			if err := InstallFiles(pkg.InstallInstructions); err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			for _, instruction := range pkg.InstallInstructions {
				fmt.Printf("(%s) -> (%s) IsDir? %t\n", instruction.Source, instruction.Destination, instruction.IsDir)
			}

			if err := database.MarkAsInstalled(pkg, pkg.InstallInstructions, pkg.Flags, db, nil, 0); err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}
		}
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove {name or id}",
	Short: "Removes a package from the system",
	Long:  "Removes a package from the system",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		GrantPrivilegies()
	},
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {

			db, err := sql.Open("sqlite3", InternalDB)
			if err != nil {
				fmt.Printf("error: %s\n", err.Error())
				os.Exit(1)
			}
			defer db.Close()

			id, err := database.GetPackageId(arg, db)
			if err != nil {
				if err == sql.ErrNoRows {
					fmt.Printf("package %s not found\n", arg)
					continue
				}
				fmt.Printf("error: %s\n", err.Error())
				continue
			}

			files, err := database.GetPackageFiles(id, db)
			if err != nil {
				fmt.Printf("error: %s\n", err.Error())
				continue
			}

			for _, file := range files {
				if !file.IsDir {
					if err := os.Remove(file.Filepath); err != nil {
						fmt.Printf("error: %s\n", err.Error())
						continue
					}
				}
			}

			if err := database.MarkAsUninstalled(id, db); err != nil {
				fmt.Printf("error removing package from database but successfully removed it from the system: %s\n", err.Error())
				continue
			}

		}
	},
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Develop a package",
	Long:  "Useful commands for developing packages",
}

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Package a directory",
	Long:  "Package a directory",
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {

			packetDotLuaBlob, err := os.ReadFile(filepath.Join(arg, "Packet.lua"))
			if err != nil {
				fmt.Printf("invalid package dir can't find Packet.lua")
				continue
			}

			packet, err := packet.ReadPacket(packetDotLuaBlob, nil)
			if err != nil {
				fmt.Printf("error: %s\n", err.Error())
				continue
			}

			packageFile, err := os.OpenFile(packet.Name+"@"+packet.Version+".pkt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Printf("error: %s\n", err.Error())
				continue
			}

			zstdWriter, err := zstd.NewWriter(packageFile)
			if err != nil {
				fmt.Printf("error: %s\n", err.Error())
				continue
			}

			defer zstdWriter.Close()

			baseDir := filepath.Clean(arg)
			tarWriter := tar.NewWriter(zstdWriter)
			defer tarWriter.Close()

			filepath.Walk(arg, func(path string, info fs.FileInfo, err error) error {

				header, err := tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}

				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return err
				}

				if relPath == "." {
					return nil
				}

				header.Name = relPath

				if err := tarWriter.WriteHeader(header); err != nil {
					return err
				}

				if !info.IsDir() {
					file, err := os.Open(path)
					if err != nil {
						return err
					}
					defer file.Close()

					if _, err := io.Copy(tarWriter, file); err != nil {
						return err
					}
				}

				return nil
			})
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed packages",
	Long:  "List all installed packages",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := sql.Open("sqlite3", InternalDB)
		if err != nil {
			panic(err)
		}
		database.PrepareDataBase(db)

		pkgs, err := database.ListAllInstalledPackages(db)
		if err != nil {
			panic(err)
		}

		for _, pkg := range pkgs {
			fmt.Printf("\033[1m==> %s\033[0m\n", pkg.Name)
			fmt.Printf("  \033[1mPackage ID:\033[0m %s\n", pkg.Id)
			fmt.Printf("  \033[1mMaintainer:\033[0m %s\n", pkg.Maintainer)
			fmt.Printf("  \033[1mVerified:\033[0m %v\n", pkg.Verified)
			fmt.Printf("  \033[1mDescription:\033[0m %s\n", pkg.Description)
			fmt.Printf("  \033[1mRepository:\033[0m %s\n", pkg.Location)
			if pkg.UploadTimeUnix == pkg.InstalledTimeUnix {
				fmt.Printf("  \033[1mInstalled timestamp:\033[0m %s\n", time.Unix(pkg.InstalledTimeUnix, 0).UTC().Local().Format("01-02-2006 15:04 Monday"))
			} else {
				fmt.Printf("  \033[1mUpload time (UTC) :\033[0m %s\n", time.Unix(pkg.InstalledTimeUnix, 0).UTC().Format("01-02-2006 15:04 Monday"))
				fmt.Printf("  \033[1mInstalled timestamp:\033[0m %s\n", time.Unix(pkg.InstalledTimeUnix, 0).UTC().Local().Format("01-02-2006 15:04 Monday"))
			}
			fmt.Printf("  \033[1mSerial:\033[0m %d\n", pkg.Serial)
			fmt.Print("\n")
		}
	},
}

var installCmd = &cobra.Command{
	Use:   "install {package name or id} ...",
	Short: "Installs a package",
	Long:  "Installs a package searching it in all repositories setted",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		GrantPrivilegies()
		return GetConfiguration()
	},
	Run: func(cmd *cobra.Command, args []string) {
		sourceDB, err := sql.Open("sqlite3", SourceDB)
		if err != nil {
			panic(err)
		}
		defer sourceDB.Close()

		internalDB, err := sql.Open("sqlite3", InternalDB)
		if err != nil {
			panic(err)
		}
		defer internalDB.Close()

		depsMap := make(map[string]map[string]repo.DependencyStatus)
		var info database.SDBPkg
		var download []string
		for _, arg := range args {

			if installed, err := database.SearchIfIsInstalled(arg, internalDB); err == nil {
				if installed {
					fmt.Printf("=> package %s is already installed\n", arg)
					continue
				}
			} else {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			info, err = database.RetrievePackageInformation(arg, "", sourceDB)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if info.Id == "" {
				fmt.Printf("error: package %s not found\n", arg)
				continue
			}
			download = append(download, PrefixForLocations+path.Join(strings.Split(info.Location, "/")[0], PrefixForPackages, string(info.Id)+".pkt"))
			if err := repo.SolveDeps(packet.PackageID(info.Id), "", internalDB, sourceDB, &depsMap); err != nil {
				panic(err)
			}
			for _, dep := range depsMap["build"] {
				strings.Split(dep.Location, "/")
				download = append(download, PrefixForLocations+path.Join(strings.Split(dep.Location, "/")[0], PrefixForPackages, string(info.Id)+".pkt"))
			}
			for _, dep := range depsMap["runtime"] {
				strings.Split(dep.Location, "/")
				download = append(download, PrefixForLocations+filepath.Join(strings.Split(dep.Location, "/")[0], PrefixForPackages, string(info.Id)+".pkt"))
			}
		}

		var wg sync.WaitGroup
		for i, url := range download {
			wg.Go(func() {
				fmt.Printf("[%d/%d] Downloading package\n", i+1, len(download))

				pkgID := packet.PackageID(path.Base(url))

				emptyReader := bytes.NewReader([]byte{})
				data := io.NopCloser(emptyReader)

				rootdir, err := filepath.Abs(filepath.Join(PackageRootDir, strings.TrimSuffix(string(pkgID), ".pkt")))
				if err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}

				if _, err := os.Stat(rootdir); err != nil {
					if os.IsNotExist(err) {
						resp, err := http.Get(url)
						if err != nil {
							fmt.Println(err)
							return
						}
						data = resp.Body
					}
				} else {
					fmt.Printf("=> Packet directory already exists checking if %s exists\n", LockFileName)
					_, err := os.Stat(filepath.Join(rootdir, LockFileName))
					if err != nil {
						if os.IsNotExist(err) {
							fmt.Printf("==> Packet directory don't have %s, the installation might be corrupted or incomplete\n", LockFileName)
							resp, err := http.Get(url)
							if err != nil {
								fmt.Println(err)
								return
							}
							data = resp.Body
						}
					} else {
						f, err := os.ReadFile(filepath.Join(rootdir, LockFileName))
						if err != nil {
							fmt.Printf("error: %s", err.Error())
							os.Exit(1)
						}
						lf := lockfile.ParseStatus(string(f))

						if !slices.Contains(lf.Progress, lockfile.Status{Action: "download", Value: url}) {
							resp, err := http.Get(url)
							if err != nil {
								fmt.Println(err)
								return
							}
							data = resp.Body
						}
					}
				}

				sourcesdir, err := filepath.Abs(filepath.Join(rootdir, "src"))
				if err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}
				configs := &packet.Config{
					BinDir:     Config.BinDir,
					RootDir:    rootdir,
					SourcesDir: sourcesdir,
				}

				db, err := sql.Open("sqlite3", InternalDB)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}
				defer db.Close()

				database.PrepareDataBase(db)

				if installed, err := database.SearchIfIsInstalled(pkgID.Name(), db); err == nil {
					if installed {
						fmt.Printf("=> package %s is already installed\n", pkgID.Name())
						return
					}
				} else {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}

				// backupDir, err := filepath.Abs(".")
				_ = ChangeToNoPermission()
				_ = os.MkdirAll(configs.RootDir, 0755)

				if err := decompress.Decompress(data, configs.RootDir, string(pkgID)); err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}
				lockFile, err := os.OpenFile(filepath.Join(rootdir, LockFileName), os.O_CREATE|os.O_RDWR, 0644)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}
				defer lockFile.Close()

				_, err = lockFile.WriteString(lockfile.NewLockfile(PacketsVersion, runtime.GOOS, runtime.GOARCH, PacketsSerial, []string{}))
				if err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}
				lockFile.WriteString("download: " + url + "\n")
				_ = os.MkdirAll(configs.SourcesDir, 0755)
				fileContent, err := os.ReadFile(filepath.Join(rootdir, "Packet.lua"))
				if err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}

				pkg, err := packet.ReadPacket(fileContent, configs)
				if err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}

				if err := DownloadSource(&pkg.GlobalSources, configs, lockFile); err != nil {
					fmt.Printf("error: %s", err.Error())
					os.Exit(1)
				}

				if pkg.Plataforms != nil {
					if plataform, exists := pkg.Plataforms[packet.OperationalSystem(runtime.GOOS)]; exists {
						if err := DownloadSource(&plataform.Sources, configs, lockFile); err != nil {
							fmt.Printf("error: %s", err.Error())
							os.Exit(1)
						}
					}
				}
			})

		}
		wg.Wait()

	},
}

var syncCmd = &cobra.Command{
	Use:   "sync {url}",
	Short: "Sync with a remote",
	Long:  "Synchronize with a remote",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		GrantPrivilegies()
		return GetConfiguration()
	},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := sql.Open("sqlite3", SourceDB)
		if err != nil {
			panic(err)
		}
		defer db.Close()
		if err := repo.FetchPackagesToDB(args[0], db); err != nil {
			panic(err)
		}
	},
}

func main() {

	verbosityLevel = os.Getenv("VERBOSE_LEVEL")

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(executeCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(flagCmd)
	rootCmd.AddCommand(listCmd)

	rootCmd.AddCommand(devCmd)
	devCmd.AddCommand(packCmd)
	rootCmd.Execute()
}
