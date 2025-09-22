package packets

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"database/sql"
	"fmt"
	"io"
	"os"
	"packets/internal/consts"
	errors_packets "packets/internal/errors"
	"packets/internal/utils"

	utils_lua "packets/internal/utils/lua"
	"path"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	lua "github.com/yuin/gopher-lua"
	_ "modernc.org/sqlite"
)

// Install exctract and fully install from a package file ( tar.zst )
func InstallPackage(file io.Reader, destDir string) error {

	manifest, err := utils.ReadManifest(file)
	if err != nil {
		return err
	}

	zstdReader, err := zstd.NewReader(file)
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
	L.SetGlobal("data_dir", lua.LFalse)
	L.SetGlobal("script", lua.LString(manifest.Hooks.Install))

	if err := L.DoFile(manifest.Hooks.Install); err != nil {
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

// GetPackage retrieves package information from the index database and downloads the package file
func GetPackage(name string) (utils.Package, error) {

	var this utils.Package
	db, err := sql.Open("sqlite", consts.IndexDB)
	if err != nil {
		return this, err
	}
	defer db.Close()

	var packageUrl string
	err = db.QueryRow("SELECT query_name, version, package_url, image_url, description, author, author_verified, os, arch, signature, public_key, family, serial, size, dependencies FROM packages WHERE name = ?", name).
		Scan(
			&this.QueryName,
			&this.Version,
			&packageUrl,
			&this.ImageUrl,
			&this.Description,
			&this.Author,
			&this.AuthorVerified,
			&this.OS,
			&this.Arch,
			&this.Signature,
			&this.PublicKey,
			&this.Family,
			&this.Serial,
			&this.Size,
			&this.Dependencies,
		)
	if err != nil {
		return utils.Package{}, err
	}

	filename := path.Base(packageUrl)
	this.Filename = filename
	peers, err := utils.AskLAN(filename)
	if err != nil {
		return utils.Package{}, err
	}

	if len(peers) > 0 {
		this.PackageF, err = utils.GetFileHTTP(packageUrl)
		if err != nil {
			return utils.Package{}, err
		}
	} else {
		var totalerrors int = 0
		for _, peer := range peers {
			this.PackageF, err = utils.GetFileHTTP(fmt.Sprintf("http://%s:%d/%s", peer.IP, peer.Port, filename))
			if err == nil {
				break
			} else {
				totalerrors++
			}
		}
		if totalerrors == len(peers) {
			this.PackageF, err = utils.GetFileHTTP(packageUrl)
			if err != nil {
				return utils.Package{}, err
			}
		}
	}

	reader := bytes.NewReader(this.PackageF)
	this.Manifest, err = utils.ReadManifest(reader)
	if err != nil {
		return utils.Package{}, err
	}

	if !ed25519.Verify(this.PublicKey, this.PackageF, this.Signature) {
		return utils.Package{}, errors_packets.ErrInvalidSignature
	}

	return this, nil
}
