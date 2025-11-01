package lua

import (
	"io"
	"os"
	"path/filepath"
)

func copyDir(src string, dest string) error {
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
					copyDir(filepath.Join(src, file.Name()), filepath.Join(dest, file.Name()))
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
			if err := copyFile(src, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(source string, destination string) error {
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
