return {
    package = {
        name = "bat-bin",
        version = "0.26.0",
        mantainer = "robogg133",
        description = "fast, opensource, easy to use package manager.",
        serial = 0,

        plataforms = {
            windows = {
                arch = {"x86_64"},
                sources = {
                    ["https://github.com/sharkdp/bat/releases/download/v0.26.0/bat-v0.26.0-aarch64-unknown-linux-gnu.tar.gz"] = {method = "GET"}
                },
                dependencies = {
                    build = {},
                    runtime = {},
                    conflicts = {}
                }
            },
            linux = {
                arch = {"x86_64"},
                sources = {
                    ["https://github.com/sharkdp/bat/releases/download/v0.26.0/bat-v0.26.0-aarch64-unknown-linux-gnu.tar.gz"] = {method = "GET"}
                },
                dependencies = {
                    build = {},
                    runtime = {},
                    conflicts = {}
                }
            },
            all = {
                sources = {}
            }
        },



    },
    
    
    prepare = function(container)
        git.clone("https://github.com/roboogg133/packets.git", container.dir("/data"))
        os.remove(container.dir("/data/.git"))

    end,
    
    build = function()
        os.execute("go build ./data/cmd/packets")
    end,
    
    install = function(container)
        os.copy(container.dir("./packets"), BIN_DIR)
    end,

    remove = function ()
        os.remove(path_join(BIN_DIR, "packets"))
    end
}