package utils

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"packets/configs"
	"packets/internal/consts"
	errors_packets "packets/internal/errors"

	"github.com/klauspost/compress/zstd"
	"github.com/pelletier/go-toml/v2"
)

type Package struct {
	PackageF       []byte
	Version        string
	ImageUrl       string
	QueryName      string
	Description    string
	Author         string
	AuthorVerified bool
	OS             string
	Arch           string
	Filename       string
	Size           int64
	Dependencies   string

	Signature []byte
	PublicKey ed25519.PublicKey

	Manifest configs.Manifest

	Family string
	Serial int
}

func GetFileHTTP(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors_packets.ErrResponseNot200OK
	}

	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

// ReadManifest is crucial to get package metadata it reads manifest.toml from a package file (tar.zst)
func ReadManifest(file io.Reader) (configs.Manifest, error) {
	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return configs.Manifest{}, err
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return configs.Manifest{}, err
		}

		if filepath.Base(header.Name) == "manifest.toml" {
			decoder := toml.NewDecoder(tarReader)

			var manifest configs.Manifest

			if err := decoder.Decode(&manifest); err != nil {
				return configs.Manifest{}, nil
			}

			return manifest, nil
		}

	}
	return configs.Manifest{}, errors_packets.ErrCantFindManifestTOML
}

// CopyDir copies a directory from source to destination
func CopyDir(src string, dest string) error {
	if stats, err := os.Stat(src); err != nil {
		return err
	} else {
		if stats.IsDir() {
			files, err := os.ReadDir(src)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}

			for _, file := range files {
				if file.IsDir() {
					CopyDir(filepath.Join(src, file.Name()), filepath.Join(dest, file.Name()))
					continue
				}
				srcFile := filepath.Join(src, file.Name())

				f, err := os.Create(filepath.Join(dest, file.Name()))
				if err != nil {
					return err
				}
				defer f.Close()

				opennedSrcFile, err := os.Open(srcFile)
				if err != nil {
					return err
				}
				defer opennedSrcFile.Close()
				if _, err := io.Copy(f, opennedSrcFile); err != nil {
					return err
				}

			}
		} else {
			if err := CopyFile(src, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyFile copies a file from source to destination
func CopyFile(source string, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	status, err := src.Stat()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(destination), 0o755)
	if err != nil {
		return err
	}

	dst, err := os.Create(destination)
	if err != nil {
		if !os.IsExist(err) {
			dst, err = os.Open(destination)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	defer dst.Close()
	if err := dst.Chmod(status.Mode()); err != nil {
		return err
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}

// Write writes the package file to the cache directory and returns the path to it
func (p *Package) Write() (string, error) {
	if err := os.WriteFile(filepath.Join(consts.DefaultCache_d, p.Filename), p.PackageF, 0o644); err != nil {
		_ = os.Remove(filepath.Join(consts.DefaultCache_d, p.Filename))
		return "", err
	}

	return filepath.Join(consts.DefaultCache_d, p.Filename), nil
}

func (p *Package) AddToInstalledDB(inCache int, packagePath string) error {
	db, err := sql.Open("sqlite", consts.InstalledDB)
	if err != nil {
		return err
	}
	defer db.Close()

	var success bool

	defer func() {
		if !success {
			_, err := db.Exec("DELETE FROM packages WHERE name = ?", p.Manifest.Info.Name)
			if err != nil {
				log.Println("Failed to rollback package addition:", err)
			}
		}
	}()

	_, err = db.Exec(`
    INSERT INTO packages (
        query_name, name, version, dependencies, description,
        family, serial, package_d, filename, os, arch, in_cache
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.QueryName,
		p.Manifest.Info.Name,
		p.Version,
		p.Dependencies,
		p.Description,
		p.Family,
		p.Serial,
		packagePath,
		p.Filename,
		p.OS,
		p.Arch,
		inCache,
	)
	if err != nil {
		return err
	}
	success = true
	return err
}

func CheckIfPackageInstalled(name string) (bool, error) {
	db, err := sql.Open("sqlite", consts.InstalledDB)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE name = ?)", name).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func GetDependencies(name string) ([]string, error) {
	db, err := sql.Open("sqlite", consts.IndexDB)
	if err != nil {
		return []string{}, err
	}
	defer db.Close()

	var dependenciesRaw string

	if err := db.QueryRow("SELECT dependencies FROM packages WHERE name = ?", name).Scan(&dependenciesRaw); err != nil {
		return []string{}, err
	}

	return strings.Fields(dependenciesRaw), nil
}

func ManifestFileRead(file io.Reader) (configs.Manifest, error) {
	decoder := toml.NewDecoder(file)

	var manifest configs.Manifest

	if err := decoder.Decode(&manifest); err != nil {
		return configs.Manifest{}, nil
	}

	return manifest, nil
}

func RemoveFromInstalledDB(name string) error {
	db, err := sql.Open("sqlite", consts.InstalledDB)
	if err != nil {
		return err
	}

	if _, err = db.Exec("DELETE FROM packages WHERE name = ?", name); err != nil {
		return err
	}

	return nil
}

// GetPackage retrieves package information from the index database and downloads the package file
func GetPackage(name string) (Package, error) {

	var this Package
	var peers []Peer

	db, err := sql.Open("sqlite", consts.IndexDB)
	if err != nil {
		return this, err
	}
	defer db.Close()

	var packageUrl string
	err = db.QueryRow("SELECT query_name, version, package_url, image_url, description, author, author_verified, os, arch, signature, public_key, family, serial, size, dependencies FROM packages WHERE name = ?", name).
		Scan(
			&this.QueryName,
			&this.Version,
			&packageUrl,
			&this.ImageUrl,
			&this.Description,
			&this.Author,
			&this.AuthorVerified,
			&this.OS,
			&this.Arch,
			&this.Signature,
			&this.PublicKey,
			&this.Family,
			&this.Serial,
			&this.Size,
			&this.Dependencies,
		)
	if err != nil {
		return Package{}, err
	}

	filename := path.Base(packageUrl)
	this.Filename = filename

	dirEntry, err := os.ReadDir(consts.DefaultCache_d)
	if err != nil {
		return Package{}, err
	}

	for _, v := range dirEntry {
		if v.Name() == filename {
			this.PackageF, err = os.ReadFile(filepath.Join(consts.DefaultCache_d, filename))
			if err != nil {
				break
			}
			goto skipping

		}
	}

	peers, err = AskLAN(filename)
	if err != nil {
		return Package{}, err
	}

	if len(peers) == 0 {
		this.PackageF, err = GetFileHTTP(packageUrl)
		if err != nil {
			return Package{}, err
		}
	} else {
		var totalerrors int = 0
		for _, peer := range peers {
			this.PackageF, err = GetFileHTTP(fmt.Sprintf("http://%s:%d/%s", peer.IP, peer.Port, filename))
			if err == nil {
				break
			} else {
				totalerrors++
			}
		}
		if totalerrors == len(peers) {
			this.PackageF, err = GetFileHTTP(packageUrl)
			if err != nil {
				return Package{}, err
			}
		}
	}

skipping:

	reader := bytes.NewReader(this.PackageF)
	this.Manifest, err = ReadManifest(reader)
	if err != nil {
		return Package{}, err
	}

	if !ed25519.Verify(this.PublicKey, this.PackageF, this.Signature) {
		return Package{}, errors_packets.ErrInvalidSignature
	}

	return this, nil
}

func GetPacketsUID() (int, error) {
	_ = exec.Command("useradd", "-M", "-N", "-r", "packets").Run()
	cmd := exec.Command("id", "-u", "packets")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return -1, err
	}

	s := strings.TrimSpace(string(out))
	uid, err := strconv.Atoi(s)
	if err != nil {
		return -1, err
	}
	return uid, nil
}

func ChangeToNoPermission() error {
	uid, err := GetPacketsUID()
	if err != nil {
		return err
	}

	return syscall.Setresuid(0, uid, 0)

}

func ElevatePermission() error { return syscall.Setresuid(0, 0, 0) }
