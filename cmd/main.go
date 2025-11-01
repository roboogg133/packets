package main

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/roboogg133/packets/pkg/install"
	"github.com/roboogg133/packets/pkg/packet.lua.d"
)

const bipath = "/usr/bin"

func main() {

	log.SetFlags(log.Llongfile)

	switch os.Args[1] {
	case "install":

		f, err := os.ReadFile(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		options := &packet.Config{
			BinDir: bipath,
		}
		pkg, err := packet.ReadPacket(f, options)
		if err != nil {
			log.Fatal(err)
		}

		pkgId := pkg.Name + "@" + pkg.Version
		if err := os.MkdirAll("_pkgtest/"+pkgId, 0777); err != nil {
			log.Fatal(err)
		}

		os.MkdirAll("_pkgtest/"+pkgId+"/src", 0777)
		os.MkdirAll("_pkgtest/"+pkgId+"/packet", 0777)
		if pkg.Plataforms != nil {
			tmp := *pkg.Plataforms

			plataform := tmp[packet.OperationalSystem(runtime.GOOS)]

			for _, v := range *plataform.Sources {
				src, err := packet.GetSource(v.Url, v.Method, v.Specs, 5)
				if err != nil {
					log.Fatal(err)
				}

				if v.Method == "GET" || v.Method == "POST" {
					f := src.([]byte)

					if err := os.WriteFile("_pkgtest/"+pkgId+"/"+path.Base(v.Url), f, 0777); err != nil {
						log.Fatal(err)
					}

					if err := packet.Dearchive("_pkgtest/"+pkgId+"/"+path.Base(v.Url), "_pkgtest/"+pkgId+"/src"); err != nil {
						log.Fatal(err)
					}
					os.Remove("_pkgtest/" + pkgId + "/" + path.Base(v.Url))

				} else {
					result, err := packet.GetSource(v.Url, v.Method, v.Specs, -213123)
					if err != nil {
						log.Fatal(err)
					}

					reponame, _ := strings.CutSuffix(path.Base(v.Url), ".git")

					_, err = git.PlainClone("_pkgtest/"+pkgId+"/src/"+reponame, result.(*git.CloneOptions))
					if err != nil {
						log.Fatal(err)
					}
				}

			}
		}
		if pkg.GlobalSources != nil {
			for _, v := range *pkg.GlobalSources {
				src, err := packet.GetSource(v.Url, v.Method, v.Specs, 5)
				if err != nil {
					log.Fatal(err)
				}

				if v.Method == "GET" || v.Method == "POST" {
					f := src.([]byte)

					if err := os.WriteFile("_pkgtest/"+pkgId+"/"+path.Base(v.Url), f, 0777); err != nil {
						log.Fatal(err)
					}

					if err := packet.Dearchive("_pkgtest/"+pkgId+"/"+path.Base(v.Url), "_pkgtest/"+pkgId+"/src"); err != nil {
						log.Fatal(err)
					}
					os.Remove("_pkgtest/" + pkgId + "/" + path.Base(v.Url))

				} else {
					result, err := packet.GetSource(v.Url, v.Method, v.Specs, -213123)
					if err != nil {
						log.Fatal(err)
					}

					reponame, _ := strings.CutSuffix(path.Base(v.Url), ".git")

					_, err = git.PlainClone("_pkgtest/"+pkgId+"/src/"+reponame, result.(*git.CloneOptions))
					if err != nil {
						log.Fatal(err)
					}
				}

			}
		}

		packetdir, err := filepath.Abs("_pkgtest/" + pkgId + "/packet")
		if err != nil {
			log.Fatal(err)
		}
		srcdir, err := filepath.Abs("_pkgtest/" + pkgId + "/src")
		if err != nil {
			log.Fatal(err)
		}
		rootdir, err := filepath.Abs("_pkgtest/" + pkgId)
		if err != nil {
			log.Fatal(err)
		}

		pkg.ExecuteBuild(&packet.Config{
			BinDir:     bipath,
			PacketDir:  packetdir,
			SourcesDir: srcdir,
			RootDir:    rootdir,
		})

		pkg.ExecuteInstall(&packet.Config{
			BinDir:     bipath,
			PacketDir:  packetdir,
			SourcesDir: srcdir,
			RootDir:    rootdir,
		})

		files, err := install.GetPackageFiles(packetdir)
		if err != nil {
			log.Fatal(err)
		}
		if err := install.InstallFiles(files, packetdir); err != nil {
			log.Fatal(err)
		}
	}
}
