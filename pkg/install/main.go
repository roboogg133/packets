package install

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

type BasicFileStatus struct {
	Filepath string
	PermMode os.FileMode
	IsDir    bool
}

func GetPackageFiles(packetDir string) ([]BasicFileStatus, error) {
	return walkAll(packetDir)
}

func walkAll(dirToWalk string) ([]BasicFileStatus, error) {
	var filesSlice []BasicFileStatus
	files, err := os.ReadDir(dirToWalk)
	if err != nil {
		return []BasicFileStatus{}, err
	}

	for _, v := range files {
		basicStat := &BasicFileStatus{
			Filepath: filepath.Join(dirToWalk, v.Name()),
			PermMode: v.Type().Perm(),
			IsDir:    v.IsDir(),
		}
		filesSlice = append(filesSlice, *basicStat)
		if v.IsDir() {
			tmp, err := walkAll(filepath.Join(dirToWalk, v.Name()))
			if err != nil {
				return []BasicFileStatus{}, err
			}
			filesSlice = append(filesSlice, tmp...)
		}
	}

	return filesSlice, nil
}

func InstallFiles(files []BasicFileStatus, packetDir string) error {
	for _, v := range files {
		sysPath, _ := strings.CutPrefix(v.Filepath, packetDir)
		if v.IsDir {
			if err := os.MkdirAll(sysPath, v.PermMode.Perm()); err != nil {
				return err
			}
		} else {
			if err := copyFile(v.Filepath, sysPath); err != nil {
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
