package utils

import (
	"archive/tar"
	"io"
	"log"
	"net/http"
	"os"
	"packets/configs"
	errors_packets "packets/internal/errors"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
	"github.com/pelletier/go-toml/v2"
)

func DownloadPackageHTTP(url string) (*[]byte, error) {

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

	return &fileBytes, nil
}

// ReadManifest is crucial to get package metadata it reads manifest.toml from a package file (tar.zst)
func ReadManifest(file *os.File) (*configs.Manifest, error) {
	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if filepath.Base(header.Name) == "manifest.toml" {
			decoder := toml.NewDecoder(tarReader)

			var manifest configs.Manifest

			if err := decoder.Decode(&manifest); err != nil {
				log.Fatal(err)
			}

			return &manifest, nil
		}

	}
	return nil, errors_packets.ErrCantFindManifestTOML
}

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
