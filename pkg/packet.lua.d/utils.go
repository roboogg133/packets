package packet

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ$%!@%&*()-=+[]{}:;.,1234567890"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func Dearchive(archive, outdir string) error {

	switch {
	case strings.HasSuffix(archive, ".tar.gz"):

		f, err := os.Open(archive)
		if err != nil {
			return err
		}

		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return err
		}

		tarReader := tar.NewReader(gzReader)

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			destination := filepath.Join(outdir, header.Name)

			fmt.Println(destination)

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.Mkdir(destination, header.FileInfo().Mode()); err != nil {
					return err
				}
			case tar.TypeReg:

				if err := os.MkdirAll(filepath.Dir(destination), 0777); err != nil {
					return err
				}
				outFile, err := os.Create(destination)
				if err != nil {
					return err
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					return err
				}
				outFile.Close()

			default:
				return fmt.Errorf("unknow filetype")
			}

		}
	}
	return nil
}
