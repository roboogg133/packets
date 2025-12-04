package lockfile

import (
	"bufio"
	"strconv"
	"strings"
)

type Lockfile struct {
	PacketsVersion string
	PacketsSerial  int
	Status         string
	TargetOS       string
	TargetArch     string
	FlagsGiven     []string
	Progress       []Status
}

type Status struct {
	Action string
	Value  string
}

const (
	VersionPrefix         = "PACKETS VERSION "
	TargetPlataformPrefix = "Target Plataform: "
	TargetArchPrefix      = "Target Architecture: "
	FlagsGivenPrefix      = "FLAGS: [ "
)

const (
	DownloadAction = "download: "
	BuildAction    = "build: "
	InstallAction  = "install: "
)

func ParseStatus(s string) Lockfile {

	lockfile := Lockfile{}

	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, VersionPrefix):
			version := strings.TrimPrefix(line, VersionPrefix)
			lockfile.PacketsVersion = strings.Split(version, " =")[0]

			lockfile.PacketsSerial, _ = strconv.Atoi(strings.TrimPrefix(strings.Split(version, " = ")[1], "SERIAL "))
			continue
		case strings.HasPrefix(line, TargetPlataformPrefix):
			lockfile.TargetOS = strings.TrimPrefix(line, TargetPlataformPrefix)
			continue
		case strings.HasPrefix(line, TargetArchPrefix):
			lockfile.TargetArch = strings.TrimPrefix(line, TargetArchPrefix)
			continue
		case strings.HasPrefix(line, FlagsGivenPrefix):
			line = strings.TrimPrefix(line, FlagsGivenPrefix)
			line = strings.TrimSuffix(line, " ]")

			lockfile.FlagsGiven = strings.Fields(line)
			continue
		}

		switch {
		case strings.HasPrefix(line, DownloadAction):
			lockfile.Progress = append(lockfile.Progress, Status{
				Action: "download",
				Value:  strings.TrimPrefix(line, DownloadAction),
			})
		case strings.HasPrefix(line, BuildAction):
			lockfile.Progress = append(lockfile.Progress, Status{
				Action: "build",
				Value:  strings.TrimPrefix(line, BuildAction),
			})
		case strings.HasPrefix(line, InstallAction):
			lockfile.Progress = append(lockfile.Progress, Status{
				Action: "install",
				Value:  strings.TrimPrefix(line, InstallAction),
			})
		}
	}
	return lockfile
}
