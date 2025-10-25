package packet

import lua "github.com/yuin/gopher-lua"

func getStringFromTable(table *lua.LTable, key string) string {
	value := table.RawGetString(key)
	if value.Type() == lua.LTString {
		return value.String()
	}
	return ""
}

func getStringArrayFromTable(L *lua.LState, table *lua.LTable, key string) []string {
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

func getDependenciesFromTable(L *lua.LState, table *lua.LTable, key string) map[string]string {
	value := table.RawGetString(key)
	if value.Type() != lua.LTTable {
		return map[string]string{}
	}

	depsTable := value.(*lua.LTable)
	dependencies := make(map[string]string)

	depsTable.ForEach(func(key, value lua.LValue) {
		if key.Type() == lua.LTString && value.Type() == lua.LTString {
			dependencies[key.String()] = value.String()
		}
	})

	return dependencies
}
