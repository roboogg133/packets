package build

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"packets/internal/consts"
	"packets/internal/packet"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/spf13/afero"
	lua "github.com/yuin/gopher-lua"
)

type Container struct {
	BuildID     BuildID
	Root        string
	FS          afero.Fs
	LuaState    lua.LState
	Manifest    packet.PacketLua
	uses        int
	DeleteAfter bool
}

func NewContainer(manifest packet.PacketLua) (Container, error) {

	var container Container
	var err error
	container.BuildID, err = getBuildId(manifest.BuildDependencies)
	if err != nil {
		return Container{}, err
	}
	baseFs := afero.NewOsFs()

	db, err := sql.Open("sqlite", consts.InstalledDB)
	if err != nil {
		return Container{}, err
	}
	if err := db.QueryRow("SELECT uses, dir FROM build_dependencies WHERE id = ? ", container.BuildID).Scan(&container.uses, container.Root); err != nil {
		db.Close()
		return Container{}, err
	}
	db.Close()

	if container.Root != "/dev/null" {
		if _, err := os.Stat(container.Root); err != nil {
			if os.IsNotExist(err) {
				if err := container.createNew(); err != nil {
					return Container{}, err
				}
			}
		}
	} else {
		container.DeleteAfter = true
		if err := container.createNew(); err != nil {
			return Container{}, err
		}
	}

	container.GetLuaState()
	fileSystem := afero.NewBasePathFs(baseFs, container.Root)

	container.Manifest = manifest
	container.FS = fileSystem

	if err := container.FS.MkdirAll(BinDir, 0777); err != nil {
		return Container{}, err
	}

	if err := container.FS.MkdirAll("/etc/packets", 0777); err != nil {
		return Container{}, err
	}

	if err := afero.Symlinker.SymlinkIfPossible(container.FS.(afero.Symlinker), BinDir, SymLinkBinDir); err != nil {
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

func getBuildId(buildDependencies map[string]string) (BuildID, error) {
	blobs, err := json.Marshal(buildDependencies)
	if err != nil {
		return "", err
	}
	return BuildID(base64.StdEncoding.EncodeToString(blobs)), nil
}

func (container Container) saveBuild() error {
	db, err := sql.Open("sqlite", consts.InstalledDB)
	if err != nil {
		return err
	}
	defer db.Close()

	buildID := container.BuildID
	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM build_dependencies WHERE id = ?)", buildID).Scan(exists); err != nil {
		return err
	}
	if exists {
		_, err := db.Exec("UPDATE FROM build_dependencies WHERE id = ? SET uses = uses + 1", buildID)
		return err
	}

	_, err = db.Exec("INSERT INTO build_dependencies (id) VALUES (?)", buildID)
	return err
}
