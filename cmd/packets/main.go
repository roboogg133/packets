package main

import (
	"archive/tar"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"packets/internal"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
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

// Install exctract and install from a package file
func Install(file *os.File) error {

	manifest, err := internal.ReadManifest(file)
	if err != nil {
		return err
	}

	name := &manifest.Info.Name

	configuration, err := internal.GetConfigTOML()
	if err != nil {
		return err
	}

	destDir := filepath.Join(configuration.Config.Data_d, *name)

	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return err
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		rel := filepath.Clean(hdr.Name)

		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		absPath := filepath.Join(destDir, rel)

		switch hdr.Typeflag {

		case tar.TypeDir:
			err = os.MkdirAll(absPath, os.FileMode(hdr.Mode))

			if err != nil {
				return err
			}

		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(absPath), 0755)
			if err != nil {
				return err
			}

			out, err := os.Create(absPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(out, tarReader)
			out.Close()
			if err != nil {
				return err
			}

			err = os.Chmod(absPath, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// COBRA CMDS

var rootCmd = &cobra.Command{Use: "packets"}
var installCmd = &cobra.Command{}
