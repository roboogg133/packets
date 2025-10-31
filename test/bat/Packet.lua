local arch_map = {
    amd64 = "x86_64",
    aarch64 = "aarch64",
    arm64 = "aarch64",
    ['386'] = "i686" 
}
local srcarch = arch_map[CURRENT_ARCH]

return {
    package = {
        name = "bat-bin", -- required
        version = "0.26.0", -- required
        maintainer = "robogg133", -- required
        description = "A cat(1) clone with syntax highlighting and Git integration.", -- required
        serial = 0,-- required

        plataforms = {
            windows = {
                arch = {"amd64"},
                sources = {
                    {
                        url = "https://github.com/sharkdp/bat/releases/download/v0.26.0/bat-v0.26.0-" ..srcarch .."-pc-windows-msvc.zip",
                        method = "GET",
                        sha256="a8a6862f14698b45e101b0932c69bc47a007f4c0456f3a129fdcef54d443d501"
                    }
                },
                dependencies = {
                    build = {},
                    runtime = {},
                    conflicts = {}
                }
            },
            linux = {
                arch = {"amd64"},
                sources = {
                    {
                        url = "https://github.com/sharkdp/bat/releases/download/v0.26.0/bat-v0.26.0-".. srcarch .."-unknown-linux-gnu.tar.gz",
                        method = "GET",
                        sha256 = "7efed0c768fae36f18ddbbb4a38f5c4b64db7c55a170dfc89fd380805809a44b"
                    }
                },
                dependencies = {
                    build = {},
                    runtime = {},
                    conflicts = {}
                }
            }
        },

        sources = {}

    },
        
    build = function() 
    end,
    
    pkg = function() -- required 
    print("oi amores")
    end,

}