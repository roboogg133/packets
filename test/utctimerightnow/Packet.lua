return {
    package = {
        name = "utctimerightnow", -- required
        version = "0.1.0", -- required
        maintainer = "robogg133", -- required
        description = "shows utc time", -- required
        serial = 0,-- required

        dependencies = {
            build = {"go"},
            runtime = {},
            conflicts = {}
        },

        sources = { --optional 
            { 
                url = "https://git.opentty.xyz/robogg133/utctimerightnow.git", -- required
                method = "git", -- required
                branch = "main" -- required 
            --  tag = ""
            }
        }

    },
        
    build = function()
     --   os.setenv("GOPATH", pathjoin(SOURCESDIR, "gopath"))
        os.chdir(pathjoin(SOURCESDIR, "utctimerightnow"))
        os.execute('go build -trimpath -ldflags="-s -w" -o utctimerightnow main.go')
        os.chmod(utctimerightnow, 0777)
    end,
    
    install  = function() -- required 
        os.copy(pathjoin(SOURCESDIR, "utctimerightnow", "utctimerightnow"), pathjoin(PACKETDIR, BIN_DIR, "utctimerightnow"))
        os.copy(pathjoin(SOURCESDIR, "utctimerightnow", "LICENSE"), pathjoin(PACKETDIR, "/usr/share/licenses/utctimerightnow/LICENSE"))
    end,

}