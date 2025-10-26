package packets

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"packets/internal/build"
	"packets/internal/consts"
	errors_packets "packets/internal/errors"
	"packets/internal/packet"
	"packets/internal/utils"
	"path"

	utils_lua "packets/internal/utils/lua"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

type Package struct {
	PackageF       []byte
	Version        string
	ImageUrl       string
	QueryName      string
	Description    string
	Author         string
	AuthorVerified bool
	OS             string
	Arch           string
	Filename       string
	Size           int64
	Dependencies   map[string]string

	Signature []byte
	PublicKey ed25519.PublicKey

	Serial int

	Manifest packet.PacketLua
}

// Install exctract and fully install from a package file ( tar.zst )
func InstallPackage(file []byte, destDir string) error {
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

		if err := os.MkdirAll(destDir, 0775); err != nil {
			return err
		}

		if err := os.Chown(destDir, uid, 0); err != nil {
			return err
		}

		absPath := filepath.Join(destDir, rel)

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

	bootstrapcontainer, err := build.NewContainer(manifest)
	if err != nil {
		return err
	}

	os.Chdir(destDir)

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

func GetPackage(id string) (Package, error) {

	var this Package
	this.Dependencies = make(map[string]string)
	var peers []Peer

	db, err := sql.Open("sqlite", consts.IndexDB)
	if err != nil {
		return this, err
	}
	defer db.Close()

	var packageUrl string
	err = db.QueryRow("SELECT query_name, version, package_url, image_url, description, author, author_verified, os, arch, signature, public_key, serial, size FROM packages WHERE id = ?", id).
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
			&this.Serial,
			&this.Size,
		)
	if err != nil {
		return Package{}, err
	}

	rows, err := db.Query("SELECT dependency_name, version_constraint FROM package_dependencies WHERE package_id = ?", id)
	if err != nil {
		return Package{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var a, vConstraint string
		if err := rows.Scan(&a, &vConstraint); err != nil {
			return Package{}, err
		}

		this.Dependencies[a] = vConstraint
	}

	filename := path.Base(packageUrl)
	this.Filename = filename

	dirEntry, err := os.ReadDir(consts.DefaultCache_d)
	if err != nil {
		return Package{}, err
	}

	for _, v := range dirEntry {
		if v.Name() == filename {
			this.PackageF, err = os.ReadFile(filepath.Join(consts.DefaultCache_d, filename))
			if err != nil {
				break
			}
			goto skipping

		}
	}

	peers, err = AskLAN(filename)
	if err != nil {
		return Package{}, err
	}

	if len(peers) == 0 {
		fmt.Printf(":: Pulling from %s\n", packageUrl)
		this.PackageF, err = getFileHTTP(packageUrl)
		if err != nil {
			return Package{}, err
		}
	} else {
		var totalerrors int = 0
		for _, peer := range peers {
			fmt.Printf(":: Pulling from local network (%s)\n", peer.IP)
			this.PackageF, err = getFileHTTP(fmt.Sprintf("http://%s:%d/%s", peer.IP, peer.Port, filename))
			if err == nil {
				break
			} else {
				totalerrors++
			}
		}
		if totalerrors == len(peers) {
			this.PackageF, err = getFileHTTP(packageUrl)
			if err != nil {
				return Package{}, err
			}
		}
	}

skipping:

	reader := bytes.NewReader(this.PackageF)
	this.Manifest, err = packet.ReadPacketFromFile(reader)
	if err != nil {
		return Package{}, err
	}

	if !ed25519.Verify(this.PublicKey, this.PackageF, this.Signature) {
		return Package{}, errors_packets.ErrInvalidSignature
	}

	return this, nil
}

func getFileHTTP(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors_packets.ErrResponseNot200OK
	}

	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}
