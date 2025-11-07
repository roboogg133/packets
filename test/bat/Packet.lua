return {
    package = {
        name = "bat-bin",                                                             -- required
        version = "0.26.0",                                                           -- required
        maintainer = "robogg133",                                                     -- required
        description = "A cat(1) clone with syntax highlighting and Git integration.", -- required
        serial = 0,
        LICENSE = { "APACHE", "MIT" },
        pageurl = "https://github.com/sharkdp/bat",

        plataforms = {
            windows = {
                arch = { "amd64" },
                sources = {
                    {
                        url = "https://github.com/sharkdp/bat/releases/download/v0.26.0/bat-v0.26.0-" ..
                            CURRENT_ARCH_NORMALIZED .. "-pc-windows-msvc.zip",
                        method = "GET",
                        sha256 = { "a8a6862f14698b45e101b0932c69bc47a007f4c0456f3a129fdcef54d443d501" }
                    }
                },
                dependencies = {
                    build = {},
                    runtime = {},
                    conflicts = {}
                }
            },
            linux = {
                arch = { "amd64" },
                sources = {
                    {
                        url = "https://github.com/sharkdp/bat/releases/download/v0.26.0/bat-v0.26.0-" ..
                            CURRENT_ARCH_NORMALIZED .. "-unknown-linux-gnu.tar.gz",
                        method = "GET",
                        sha256 = { "7efed0c768fae36f18ddbbb4a38f5c4b64db7c55a170dfc89fd380805809a44b" }
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

    install = function()
        os.chdir(pathjoin(SOURCESDIR, "bat-v0.26.0-" .. CURRENT_ARCH_NORMALIZED .. "-unknown-linux-gnu"))
        os.chmod("bat", 755)

        install("bat", pathjoin(BIN_DIR, "bat"))
        install("bat.1", pathjoin("/usr/share/man/man1/bat.1"))

        setflags("man", "manual", "/usr/share/man/man1/bat.1")
    end,

}
