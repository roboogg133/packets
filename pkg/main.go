package pkg

import (
	"archive/tar"
	"io"
	"os"
	"packets/configs"
	"packets/internal/utils"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// Install exctract and install from a package file ( tar.zst )
func InstallPackage(file *os.File) error {

	manifest, err := utils.ReadManifest(file)
	if err != nil {
		return err
	}

	name := &manifest.Info.Name

	configuration, err := configs.GetConfigTOML()
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
