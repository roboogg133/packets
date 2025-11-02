package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/roboogg133/packets/pkg/install"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
	"github.com/spf13/cobra"
)

var Config *PacketsConfiguration

var rootCmd = &cobra.Command{
	Use:   "packets",
	Short: "A tool for managing packets",
	Long:  "A multiplatform package manager",
}

var executeCmd = &cobra.Command{
	Use:   "execute {path}",
	Short: "Installs a package from a given Packet.lua file",
	Long:  "Installs a package from a given Packet.lua file",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return GetConfiguration()
	},
	Run: func(cmd *cobra.Command, args []string) {

		for _, v := range args {
			if !strings.HasSuffix(v, ".lua") {
				fmt.Printf("error: %s need to have .lua suffix\n", v)
				os.Exit(1)
			}
			contentBlob, err := os.ReadFile(v)
			if err != nil {
				fmt.Printf("error: %s could not be read\n", v)
				os.Exit(1)
			}
			pkg, err := packet.ReadPacket(contentBlob, &packet.Config{BinDir: Config.BinDir})
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

			_ = os.MkdirAll(configs.RootDir, 0755)
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
			backupDir, err := filepath.Abs(".")
			if err != nil {
				fmt.Printf("error: %s", err.Error())
				os.Exit(1)
			}
			pkg.ExecuteBuild(configs)
			pkg.ExecuteInstall(configs)
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

		}
	},
}

func main() {
	rootCmd.AddCommand(executeCmd)
	rootCmd.Execute()
}
