package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

func InstallFiles(instructions []packet.InstallInstruction) error {

	for _, v := range instructions {

		if v.IsDir {
			if err := os.MkdirAll(v.Destination, v.FileMode); err != nil {
				return err
			}
		} else {
			if err := copyFile(v.Source, v.Destination); err != nil {
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
