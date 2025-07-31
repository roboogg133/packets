package internal

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ulikunitz/xz"
)

type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Author       string   `json:"author"`
	Family       string   `json:"family"`
	Serial       uint     `json:"serial"`
}

func PacketsPackageDir() string {

	out, _ := exec.Command("uname", "-s").Output()

	if uname := strings.TrimSpace(string(out)); uname == "OpenTTY" {
		return "/mnt/....."
	} else {
		return "/etc/packets"
	}
}

func ManifestReadXZ(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(xzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(hdr.Name, "/manifest.json") || hdr.Name == "manifest.json" {

			var manifest Manifest
			decoder := json.NewDecoder(tr)
			if err := decoder.Decode(&manifest); err != nil {
				return nil, err
			}
			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("can't find manifest.json")
}
