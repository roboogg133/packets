package decompress

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/ulikunitz/xz"
)

func extractZipFile(file *zip.File, dest string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(dest, file.Name)

	if file.FileInfo().IsDir() {
		return os.MkdirAll(path, file.Mode())
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

func Decompress(data []byte, outputDir, filename string) error {

	var reader io.Reader
	switch {
	case strings.HasSuffix(filename, ".gz"):
		var err error
		reader, err = gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return err
		}
		filename, _ = strings.CutSuffix(filename, ".gz")
	case strings.HasSuffix(filename, ".xz"):
		var err error
		reader, err = xz.NewReader(bytes.NewReader(data))
		if err != nil {
			return err
		}
		filename, _ = strings.CutSuffix(filename, ".xz")
	case strings.HasSuffix(filename, ".zst"):
		var err error
		reader, err = zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			return err
		}
		filename, _ = strings.CutSuffix(filename, ".zst")
	case strings.HasSuffix(filename, ".bz2"):
		reader = bzip2.NewReader(bytes.NewReader(data))
		filename, _ = strings.CutSuffix(filename, ".bz2")
	case strings.HasSuffix(filename, ".lz4"):
		reader = lz4.NewReader(bytes.NewReader(data))
		filename, _ = strings.CutSuffix(filename, ".lz4")
	case strings.HasSuffix(filename, ".zip"):
		byteReader := bytes.NewReader(data)
		reader, err := zip.NewReader(byteReader, int64(len(data)))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return err
		}

		for _, file := range reader.File {
			err := extractZipFile(file, outputDir)
			if err != nil {
				return fmt.Errorf("error unziping %s: %w", file.Name, err)
			}
		}
		return nil
	}

	if strings.HasSuffix(filename, ".tar") {

		tarReader := tar.NewReader(reader)

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			targetPath := filepath.Join(outputDir, filepath.Clean(header.Name))
			if !strings.HasPrefix(targetPath, outputDir) {
				return fmt.Errorf("invalid path: %s", targetPath)
			}

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
					return err
				}

			case tar.TypeReg, tar.TypeRegA:
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return err
				}

				outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
				defer outFile.Close()

				if _, err := io.Copy(outFile, tarReader); err != nil {
					return err
				}

			case tar.TypeSymlink:
				if err := os.Symlink(header.Linkname, targetPath); err != nil {
					return err
				}
			case tar.TypeLink:
				linkPath := filepath.Join(outputDir, header.Linkname)
				if err := os.Link(linkPath, targetPath); err != nil {
					return err
				}

			default:
				return fmt.Errorf("unknown file type: %c => %s", header.Typeflag, header.Name)
			}
		}
	}

	return nil
}
