package build

import (
	"io"
	"path/filepath"

	"github.com/spf13/afero"
)

func (container Container) copyContainer(src string, dest string) error {
	stats, err := container.FS.Stat(src)
	if err != nil {
		return err
	}

	if stats.IsDir() {
		files, err := afero.ReadDir(container.FS, src)
		if err != nil {
			return err
		}

		if err := container.FS.MkdirAll(dest, 0755); err != nil {
			return err
		}

		for _, file := range files {
			srcPath := filepath.Join(src, file.Name())
			destPath := filepath.Join(dest, file.Name())

			if file.IsDir() {
				if err := container.CopyHostToContainer(srcPath, destPath); err != nil {
					return err
				}
				continue
			}

			if err := container.copySingleFileLtoL(srcPath, destPath); err != nil {
				return err
			}
		}
	} else {
		if err := container.copySingleFileLtoL(src, dest); err != nil {
			return err
		}
	}

	return nil
}

func (container Container) copySingleFileLtoL(source string, destination string) error {
	src, err := container.FS.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	stats, err := src.Stat()
	if err != nil {
		return err
	}
	if err := container.FS.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	dst, err := container.FS.Create(destination)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	if err := container.FS.Chmod(destination, stats.Mode()); err != nil {
		return err
	}

	return nil
}
