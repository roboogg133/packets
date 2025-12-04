package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/roboogg133/packets/cmd/packets/decompress"
	"github.com/roboogg133/packets/cmd/packets/lockfile"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

func DownloadSource(sources *[]packet.Source, configs *packet.Config, lockFile *os.File) error {
	for _, source := range *sources {
		if lockFile != nil {
			b, err := io.ReadAll(lockFile)
			if err != nil {
				return fmt.Errorf("error: %s", err.Error())
			}
			lf := lockfile.ParseStatus(string(b))
			if slices.Contains(lf.Progress, lockfile.Status{Action: "download", Value: source.Url}) {
				fmt.Printf("===> Skipping download %s\n", path.Base(source.Url))
				continue
			}
		}
		downloaded, err := packet.GetSource(source.Url, source.Method, source.Specs, NumberOfTryAttempts)
		if err != nil {
			return fmt.Errorf("error: %s", err.Error())
		}
		if source.Method == "GET" || source.Method == "POST" {
			f := downloaded.([]byte)

			buf := bytes.NewBuffer(f)
			_ = os.MkdirAll(configs.SourcesDir, 0755)

			if err := decompress.Decompress(buf, configs.SourcesDir, path.Base(source.Url)); err != nil {
				return fmt.Errorf("error: %s", err.Error())
			}
		} else {
			options := downloaded.(*git.CloneOptions)
			repoName, _ := strings.CutSuffix(filepath.Base(source.Url), ".git")
			_ = os.MkdirAll(filepath.Join(configs.SourcesDir, repoName), 0755)
			_, err := git.PlainClone(filepath.Join(configs.SourcesDir, repoName), options)
			if err != nil {
				return fmt.Errorf("error: %s", err.Error())
			}
			os.RemoveAll(filepath.Join(configs.SourcesDir, repoName, ".git"))
		}
		fmt.Printf("===> Download: %s\n", path.Base(source.Url))
		if lockFile != nil {
			lockFile.WriteString("download: " + source.Url + "\n")
		}
	}
	return nil
}
