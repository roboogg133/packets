package build

import (
	"io"
	"os"
	"packets/configs"
	utils_lua "packets/internal/utils/lua"
	"path/filepath"

	"github.com/spf13/afero"
	lua "github.com/yuin/gopher-lua"
)

type Container struct {
	Root     string
	FS       afero.Fs
	DataDir  string
	Manifest configs.Manifest
}

func NewContainer(Root string, dataDir string, manifest configs.Manifest) (Container, error) {

	var container Container
	baseFs := afero.NewOsFs()
	fileSystem := afero.NewBasePathFs(baseFs, Root)

	container.Root = Root
	container.Manifest = manifest
	container.DataDir = dataDir
	container.FS = fileSystem

	if err := container.CopyHostToContainer(dataDir, "/data"); err != nil {
		return Container{}, err
	}

	if err := container.FS.MkdirAll("/usr/bin", 0777); err != nil {
		return Container{}, err
	}

	if err := afero.Symlinker.SymlinkIfPossible(container.FS.(afero.Symlinker), "/usr/bin", "/bin"); err != nil {
		return Container{}, err
	}

	return container, nil
}

func (container Container) CopyHostToContainer(src string, dest string) error {
	stats, err := os.Stat(src)
	if err != nil {
		return err
	}

	if stats.IsDir() {
		files, err := os.ReadDir(src)
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

			if err := container.copySingleFile(srcPath, destPath); err != nil {
				return err
			}
		}
	} else {
		if err := container.copySingleFile(src, dest); err != nil {
			return err
		}
	}

	return nil
}

func (container Container) copySingleFile(source string, destination string) error {
	src, err := os.Open(source)
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

func (container Container) RunBuild() error {

	L, err := utils_lua.GetSandBox()
	if err != nil {
		return err
	}

	L.SetGlobal("data_dir", lua.LString(container.DataDir))
	L.SetGlobal("script", lua.LString(container.Manifest.Hooks.Build))

	return nil
}
