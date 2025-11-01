package packet

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func getStringFromTable(table *lua.LTable, key string) string {
	value := table.RawGetString(key)
	if value.Type() == lua.LTString {
		return value.String()
	}
	return ""
}

func getIntFromTable(table *lua.LTable, key string) int {
	value := table.RawGetString(key)
	if value.Type() == lua.LTNumber {
		if num, ok := value.(lua.LNumber); ok {
			return int(num)
		}
	}
	return -133
}

func getStringArrayFromTable(table *lua.LTable, key string) []string {
	value := table.RawGetString(key)
	if value.Type() != lua.LTTable {
		return []string{}
	}

	arrayTable := value.(*lua.LTable)
	var result []string

	arrayTable.ForEach(func(_, value lua.LValue) {
		if value.Type() == lua.LTString {
			result = append(result, value.String())
		}
	})

	return result
}

func getFunctionFromTable(table *lua.LTable, key string) *lua.LFunction {
	value := table.RawGetString(key)
	if value.Type() == lua.LTFunction {
		return value.(*lua.LFunction)
	}
	return nil
}

type version struct {
	Name       string
	Constraint VersionConstraint
}

func getDependenciesFromTable(table *lua.LTable, key string) *PkgDependencies {
	value := table.RawGetString(key)
	if value.Type() != lua.LTTable {
		return &PkgDependencies{}
	}

	var pkgDeps PkgDependencies

	depnTable := value.(*lua.LTable)

	pkgDeps.RuntimeDependencies = depsParse(depnTable, "runtime")
	pkgDeps.BuildDependencies = depsParse(depnTable, "build")
	pkgDeps.Conflicts = depsParse(depnTable, "conflicts")

	return &pkgDeps
}

func getSourcesFromTable(table *lua.LTable, key string) *[]Source {
	value := table.RawGetString(key)
	if value.Type() != lua.LTTable {
		return nil
	}

	var srcList []Source

	srcTable := value.(*lua.LTable)

	srcTable.ForEach(func(_, value lua.LValue) {
		if value.Type() == lua.LTTable {
			src := value.(*lua.LTable)

			var srcInfo Source

			method := src.RawGetString("method")
			if method.Type() == lua.LTString {
				srcInfo.Method = method.String()
			}

			url := src.RawGetString("url")

			if url.Type() == lua.LTString {
				srcInfo.Url = url.String()
			}

		switchlabel:
			switch srcInfo.Method {
			case "GET":
				var getSpecs GETSpecs

				getSpecs.SHA256 = new(string)
				sha256sumL := src.RawGetString("sha256")
				if sha256sumL.Type() == lua.LTString {
					*getSpecs.SHA256 = sha256sumL.String()
				}

				headersLT := src.RawGetString("headers")
				if headersLT.Type() == lua.LTTable {

					headers := headersLT.(*lua.LTable)

					tmpMap := make(map[string]string)
					headers.ForEach(func(headerKey, value lua.LValue) {
						if value.Type() == lua.LTString {
							tmpMap[headerKey.String()] = value.String()
						}
					})

					getSpecs.Headers = &tmpMap
				}
				srcInfo.Specs = getSpecs
				break switchlabel

			case "git":
				var gitSpecs GitSpecs

				branchL := src.RawGetString("branch")

				if branchL.Type() == lua.LTString {
					gitSpecs.Branch = branchL.String()
				}

				tagL := src.RawGetString("tag")

				if tagL.Type() == lua.LTString {
					*gitSpecs.Tag = tagL.String()
				}

				srcInfo.Specs = gitSpecs
				break switchlabel
			case "POST":
				var postSpecs POSTSpecs

				sha256sumL := src.RawGetString("sha256")
				if sha256sumL.Type() == lua.LTString {
					*postSpecs.SHA256 = sha256sumL.String()
				}

				headersLT := src.RawGetString("headers")
				if headersLT.Type() == lua.LTTable {

					headers := headersLT.(*lua.LTable)

					tmpMap := make(map[string]string)
					headers.ForEach(func(headerKey, value lua.LValue) {
						if value.Type() == lua.LTString {
							tmpMap[headerKey.String()] = value.String()
						}
					})

					postSpecs.Headers = &tmpMap
				}

				bodyLt := src.RawGetString("body")

				if bodyLt.Type() == lua.LTString {
					*postSpecs.Body = bodyLt.String()
				}

				srcInfo.Specs = postSpecs
				break switchlabel
			}
			srcList = append(srcList, srcInfo)
		}
	})

	return &srcList
}

func getPlataformsFromTable(table *lua.LTable, key string) *map[OperationalSystem]Plataform {
	value := table.RawGetString(key)

	if value.Type() != lua.LTTable {
		return nil
	}

	tmpMap := make(map[OperationalSystem]Plataform)

	plataform := value.(*lua.LTable)

	plataform.ForEach(func(osString, value lua.LValue) {
		if value.Type() != lua.LTTable {
			return
		}

		var plat Plataform
		plat.Architetures = getStringArrayFromTable(value.(*lua.LTable), "arch")
		plat.Name = osString.String()
		plat.Sources = getSourcesFromTable(value.(*lua.LTable), "sources")
		plat.Dependencies = getDependenciesFromTable(value.(*lua.LTable), "dependencies")

		tmpMap[OperationalSystem(osString.String())] = plat
	})

	if len(tmpMap) == 0 {
		return nil
	}

	return &tmpMap
}

func depsParse(depnTable *lua.LTable, key string) *map[string]*VersionConstraint {
	if runLTable := depnTable.RawGetString(key); runLTable.Type() == lua.LTTable {
		runtimeTable := runLTable.(*lua.LTable)

		mapTemp := make(map[string]*VersionConstraint)

		var found bool

		runtimeTable.ForEach(func(_, value lua.LValue) {
			if value.Type() == lua.LTString {
				version := parseVersionString(value.String())
				mapTemp[version.Name] = &version.Constraint
				found = true
			}
		})
		if !found {
			return nil
		} else {
			return &mapTemp
		}

	}
	return nil
}

func parseVersionString(s string) version {
	// >=go@1.25.3 | <=go@1.25.3 | go | >go@1.25.3 | <go@1.25.3 | go@1.25.3
	if strings.ContainsAny(s, "@") {
		slice := strings.Split(s, "@")

		switch {
		case !strings.ContainsAny(s, "<=>"):
			return version{
				Name:       slice[0],
				Constraint: VersionConstraint(slice[1]),
			}
		case s[0] == '>' && s[1] == '=':

			return version{
				Name:       slice[0][2:],
				Constraint: VersionConstraint(">=" + slice[1]),
			}
		case s[0] == '<' && s[1] == '=':

			return version{
				Name:       slice[0][2:],
				Constraint: VersionConstraint("<=" + slice[1]),
			}

		case s[0] == '>' && s[1] != '=':

			return version{
				Name:       slice[0][1:],
				Constraint: VersionConstraint(">" + slice[1]),
			}
		case s[0] == '<' && s[1] != '=':

			return version{
				Name:       slice[0][1:],
				Constraint: VersionConstraint("<" + slice[1]),
			}
		}

	} else if !strings.ContainsAny(s, "@=<>") {
		return version{
			Name:       s,
			Constraint: VersionConstraint(0x000),
		}
	}

	return version{}
}

func normalizeArch(arch string) string {
	switch arch {
	case "386":
		return "i686"
	case "amd64":
		return "x86_64"
	case "amd64p32":
		return "x86_64"
	case "arm":
		return "arm"
	case "arm64":
		return "aarch64"
	case "arm64be":
		return "aarch64_be"
	case "armbe":
		return "armbe"
	case "loong64":
		return "loongarch64"
	case "mips":
		return "mips"
	case "mips64":
		return "mips64"
	case "mips64le":
		return "mips64el"
	case "mips64p32":
		return "mips64"
	case "mips64p32le":
		return "mips64el"
	case "mipsle":
		return "mipsel"
	case "ppc":
		return "powerpc"
	case "ppc64":
		return "ppc64"
	case "ppc64le":
		return "ppc64le"
	case "riscv":
		return "riscv"
	case "riscv64":
		return "riscv64"
	case "s390":
		return "s390"
	case "s390x":
		return "s390x"
	case "sparc":
		return "sparc"
	case "sparc64":
		return "sparc64"
	case "wasm":
		return "wasm"
	default:
		return arch
	}
}
