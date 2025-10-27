return {
    package = {
        name = "packets",
        id   = "packets@1.0.0",
        version = "1.0.0",
        author  = "robogg133",
        description = "fast, opensource, easy to use package manager.",
        type = "remote",
        serial = 0,

        build_dependencies = {["go"] = ">=1.25.1"},

        git_url = "https://github.com/roboogg133/packets.git",
        git_branch = "main"

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