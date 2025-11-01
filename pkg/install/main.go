package install

import (
	"fmt"
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
			filesSlice, err = walkAll(filepath.Join(dirToWalk, v.Name()))
			if err != nil {
				return []BasicFileStatus{}, err
			}
		}
	}

	return filesSlice, nil
}

func InstallFiles(files []BasicFileStatus, packetDir string) error {
	for i, v := range files {
		sysPath, _ := strings.CutPrefix(v.Filepath, packetDir)
		fmt.Printf("[%d] Installing file %s\n", i, v.Filepath)
		fmt.Printf("[%d] NEED tro track file %s\n", i, sysPath)
		if err := copyFile(v.Filepath, sysPath); err != nil {
			return err
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
