package build

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"packets/internal/packet"
	"packets/internal/utils"
	utils_lua "packets/internal/utils/lua"
	"path/filepath"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
)

func (container Container) installPackage(file []byte, destDir string) error {
	manifest, err := packet.ReadPacketFromFile(bytes.NewReader(file))
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

			if filepath.Base(hdr.Name) == "Packet.lua" {
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

	bootstrapcontainer, err := NewContainer(manifest)
	if err != nil {
		return err
	}

	if err := bootstrapcontainer.ExecutePrepare(manifest, &L); err != nil {
		return fmt.Errorf("error executing prepare: %s", err)
	}

	if err := bootstrapcontainer.ExecuteBuild(manifest, &L); err != nil {
		return fmt.Errorf("error executing build: %s", err)
	}

	if err := utils.ChangeToNoPermission(); err != nil {
		return fmt.Errorf("error changing to packet user: %s", err)
	}
	if err := bootstrapcontainer.ExecuteInstall(manifest, &L); err != nil {
		return fmt.Errorf("error executing build: %s", err)
	}

	if err := utils.ElevatePermission(); err != nil {
		return fmt.Errorf("error changing to root: %s", err)
	}

	return nil
}

func (container Container) asyncFullInstallDependencie(dep string, storePackages bool, installPath string, wg *sync.WaitGroup) {
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
