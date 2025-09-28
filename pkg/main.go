package packets

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"packets/internal/utils"

	utils_lua "packets/internal/utils/lua"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	lua "github.com/yuin/gopher-lua"
	_ "modernc.org/sqlite"
)

// Install exctract and fully install from a package file ( tar.zst )
func InstallPackage(file []byte, destDir string) error {
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

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		absPath := filepath.Join(destDir, rel)

		switch hdr.Typeflag {

		case tar.TypeDir:
			err = os.MkdirAll(absPath, os.FileMode(hdr.Mode))

			if err != nil {
				return err
			}

		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(absPath), 0755)
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

			err = os.Chmod(absPath, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		}
	}

	L, err := utils_lua.GetSandBox(destDir)
	if err != nil {
		return err
	}
	L.SetGlobal("DATA_DIR", lua.LString(filepath.Join(destDir, "data")))
	L.SetGlobal("script", lua.LString(manifest.Hooks.Install))

	if err := L.DoFile(filepath.Join(destDir, manifest.Hooks.Install)); err != nil {
		return err
	}

	return nil
}

// ExecuteRemoveScript executes the remove script from the package
func ExecuteRemoveScript(path string) error {

	L, err := utils_lua.GetSandBox(".")
	if err != nil {
		return err
	}

	L.SetGlobal("data_dir", lua.LFalse)
	L.SetGlobal("script", lua.LString(path))
	L.SetGlobal("build", lua.LNil)

	if err := L.DoFile(path); err != nil {
		return err
	}

	return nil
}
