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
