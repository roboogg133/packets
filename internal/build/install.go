package build

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"packets/internal/utils"
	utils_lua "packets/internal/utils/lua"
	"path/filepath"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
	lua "github.com/yuin/gopher-lua"
)

func (container Container) installPackage(file []byte, destDir string) error {
	manifest, err := utils.ReadManifest(bytes.NewReader(file))
	if err != nil {
		return err
	}

	zstdReader, err := zstd.NewReader(bytes.NewReader(file))
	if err != nil {
		return err
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)

	uid, err := utils.GetPacketsUID()
	if err != nil {
		return err
	}

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

		if err := os.MkdirAll(filepath.Join(container.Root, destDir), 0775); err != nil {
			return err
		}

		if err := os.Chown(filepath.Join(container.Root, destDir), uid, 0); err != nil {
			return err
		}

		absPath := filepath.Join(filepath.Join(container.Root, destDir), rel)

		switch hdr.Typeflag {

		case tar.TypeDir:
			err = os.MkdirAll(absPath, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if err := os.Chown(absPath, uid, 0); err != nil {
				return err
			}

		case tar.TypeReg:

			err = os.MkdirAll(filepath.Dir(absPath), 0775)
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

			err = os.Chmod(absPath, os.FileMode(0775))
			if err != nil {
				return err
			}

			if filepath.Base(hdr.Name) == "manifest.toml" || filepath.Base(hdr.Name) == manifest.Hooks.Install || filepath.Base(hdr.Name) == manifest.Hooks.Remove {
				err = os.Chmod(absPath, os.FileMode(0755))
				if err != nil {
					return err
				}
			} else {
				if err := os.Chown(absPath, uid, 0); err != nil {
					return err
				}
			}
		}
	}

	L, err := utils_lua.GetSandBox()
	if err != nil {
		return err
	}

	L.SetGlobal("DATA_DIR", lua.LString(filepath.Join(destDir, "data")))
	L.SetGlobal("script", lua.LString(manifest.Hooks.Build))

	bootstrapcontainer, err := NewContainer(filepath.Join(container.Root, destDir, "data"), manifest)
	if err != nil {
		return err
	}

	bootstrapcontainer.LuaState.DoFile(manifest.Hooks.Build)

	L.SetGlobal("DATA_DIR", lua.LString(filepath.Join(destDir, "data")))
	L.SetGlobal("script", lua.LString(manifest.Hooks.Install))

	if err := utils.ChangeToNoPermission(); err != nil {
		return err
	}
	if err := L.DoFile(filepath.Join(destDir, manifest.Hooks.Install)); err != nil {
		return err
	}

	if err := utils.ElevatePermission(); err != nil {
		return err
	}

	return nil
}

func (container Container) asyncFullInstallDependencie(dep string, storePackages bool, installPath string, wg *sync.WaitGroup, mu *sync.Mutex) {
	defer wg.Done()

	fmt.Printf(" downloading %s \n", dep)
	p, err := utils.GetPackage(dep)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(" installing %s \n", dep)

	if err := container.installPackage(p.PackageF, installPath); err != nil {
		log.Fatal(err)
	}

	if storePackages {
		_, err := p.Write()
		if err != nil {
			log.Fatal(err)
			return
		}
	}
}
