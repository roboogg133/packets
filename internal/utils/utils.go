package utils

import (
	"archive/tar"
	"crypto/ed25519"
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"packets/configs"
	"packets/internal/consts"
	errors_packets "packets/internal/errors"
	"path/filepath"

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
				log.Fatal(err)
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

			if err := os.MkdirAll(dest, 0755); err != nil {
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

	err = os.MkdirAll(filepath.Dir(destination), 0755)
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

	if err := os.WriteFile(filepath.Join(consts.DefaultCache_d, p.Filename), p.PackageF, 0644); err != nil {
		_ = os.Remove(filepath.Join(consts.DefaultCache_d, p.Filename))
		return "", err
	}

	return filepath.Join(consts.DefaultCache_d, p.Filename), nil
}

func (p *Package) AddToInstalledDB() error {
	db, err := sql.Open("sqlite", consts.InstalledDB)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO packages (name, version, dependencies)")
	return err
}
