package main

import (
	"archive/tar"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zstd"
	_ "github.com/mattn/go-sqlite3"
	"github.com/roboogg133/packets/cmd/packets/database"
	"github.com/roboogg133/packets/cmd/packets/decompress"
	"github.com/roboogg133/packets/pkg/install"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
	"github.com/spf13/cobra"
)

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
			packetsdir, err := filepath.Abs(filepath.Join(rootdir, "packet"))
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}
			configs := &packet.Config{
				BinDir:     Config.BinDir,
				RootDir:    rootdir,
				SourcesDir: sourcesdir,
				PacketDir:  packetsdir,
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
					fmt.Printf("package %s is already installed", pkg.Name)
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
			_ = os.MkdirAll(configs.PacketDir, 0755)

			if err := DownloadSource(pkg.GlobalSources, configs); err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			if pkg.Plataforms != nil {

				temp := *pkg.Plataforms

				if plataform, exists := temp[packet.OperationalSystem(runtime.GOOS)]; exists {
					if err := DownloadSource(plataform.Sources, configs); err != nil {
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

			files, err := install.GetPackageFiles(configs.PacketDir)
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			if err := install.InstallFiles(files, configs.PacketDir); err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}

			if err := database.MarkAsInstalled(pkg, files, configs.PacketDir, pkg.Flags, db, nil, 0); err != nil {
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

			if err := database.MarkAsUninstalled(id.ID, db); err != nil {
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

func main() {
	rootCmd.AddCommand(executeCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(configCmd)

	rootCmd.AddCommand(devCmd)
	devCmd.AddCommand(packCmd)
	rootCmd.Execute()
}
