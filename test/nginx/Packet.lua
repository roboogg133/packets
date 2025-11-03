-- https://nginx.org/download/nginx-1.29.3.tar.gz
return {
    package = {
        name = "nginx",
        version = "1.29.3",
        maintainer = "robogg133",
        description =
        [[nginx ("engine x") is an HTTP web server, reverse proxy, content cache, load balancer, TCP/UDP proxy server, and mail proxy server. Originally written by Igor Sysoev and distributed under the 2-clause BSD License. Enterprise distributions, commercial support and training are available from F5, Inc.]],
        serial = 0,

        plataforms = {
            windows = {
                arch = { "amd64" },
                sources = {
                    {
                        url = "https://nginx.org/download/nginx-1.29.3.zip",
                        method = "GET",
                        sha256 = { "afa2fde9fdf0ac64b91a17dcd34100ac557a3ff8e6154eeb0eeae7aa8e5bbc2d" }
                    }
                },
                dependencies = {
                    build = {
                        "cc",
                        "cmake",
                        "make"
                    },
                    runtime = {},
                    conflicts = {}
                }
            },
            linux = {
                arch = { "amd64" },
                sources = {
                    {
                        url = "https://nginx.org/download/nginx-1.29.3.tar.gz",
                        method = "GET",
                        sha256 = { "9befcced12ee09c2f4e1385d7e8e21c91f1a5a63b196f78f897c2d044b8c9312" }

                    }
                },
                dependencies = {
                    build = {
                        "cc",
                        "cmake",
                        "make"
                    },
                    runtime = {},
                    conflicts = {}
                }
            }
        },

        sources = {}

    },

    build = function()
        local uncompressedname = "nginx-1.29.3"

        os.chdir(pathjoin(SOURCESDIR, uncompressedname))
        os.chmod("configure", 0755)
        os.execute("./configure --prefix=/etc/nginx --conf-path=/etc/nginx/nginx.conf --sbin-path=" ..
            pathjoin(BIN_DIR, "nginx"))

        print("Build progress: executing Make...")
        local handle = io.popen("make", "r")
        local _ = handle:read("*a")
        local success, reason, exitcode = handle:close()

        if not success then
            error("make failed with code " .. tostring(exitcode) .. ": " .. tostring(reason))
        end
        print("Build progress: Make completed!")
    end,

    install = function()
        local uncompressedname = "nginx-1.29.3"

        os.copy("nginx.service", pathjoin(PACKETDIR, "/etc/systemd/system/nginx.service"))
        os.chdir(pathjoin(SOURCESDIR, uncompressedname))

        os.chmod("objs/nginx", 755)
        os.copy("objs/nginx", pathjoin(PACKETDIR, BIN_DIR, "nginx"))
        os.mkdir(pathjoin(PACKETDIR, "/usr/local/nginx"), 755)
        os.mkdir(pathjoin(PACKETDIR, "/etc/nginx"), 755)

        os.copy("conf/koi-win", pathjoin(PACKETDIR, "/etc/nginx/koi-win"))
        os.copy("conf/koi-utf", pathjoin(PACKETDIR, "/etc/nginx/koi-utf"))
        os.copy("conf/win-utf", pathjoin(PACKETDIR, "/etc/nginx/win-utf"))

        os.copy("conf/mime.types", pathjoin(PACKETDIR, "/etc/nginx/mime.types"))
        os.copy("conf/mime.types", pathjoin(PACKETDIR, "/etc/nginx/mime.types.default"))

        os.copy("conf/fastcgi_params", pathjoin(PACKETDIR, "/etc/nginx/fastcgi_params"))
        os.copy("conf/fastcgi_params", pathjoin(PACKETDIR, "/etc/nginx/fastcgi_params.default"))

        os.copy("conf/fastcgi.conf", pathjoin(PACKETDIR, "/etc/nginx/fastcgi.conf"))
        os.copy("conf/fastcgi.conf", pathjoin(PACKETDIR, "/etc/nginx/fastcgi.conf.default"))

        os.copy("conf/uwsgi_params", pathjoin(PACKETDIR, "/etc/nginx/uwsgi_params"))
        os.copy("conf/uwsgi_params", pathjoin(PACKETDIR, "/etc/nginx/uwsgi_params.default"))


        os.copy("conf/scgi_params", pathjoin(PACKETDIR, "/etc/nginx/scgi_params"))
        os.copy("conf/scgi_params", pathjoin(PACKETDIR, "/etc/nginx/scgi_params.default"))

        os.copy("conf/nginx.conf", pathjoin(PACKETDIR, "/etc/nginx/nginx.conf"))
        os.copy("conf/nginx.conf", pathjoin(PACKETDIR, "/etc/nginx/nginx.conf.default"))

        os.copy("html", pathjoin(PACKETDIR, "/usr/share/nginx/html"))

        os.copy("LICENSE", pathjoin(PACKETDIR, "/usr/share/licenses/nginx/LICENSE"))

        os.copy("man/nginx.8", pathjoin(PACKETDIR, "/usr/share/man/man8/nginx.8"))

        os.mkdir(pathjoin(PACKETDIR, "/etc/nginx/logs"), 755)

        setflags("bin", "nginx", pathjoin(BIN_DIR, "nginx"))
        setflags("config", "main", "/etc/nginx/nginx.conf")
        setflags("config", "sites-available", "/etc/nginx/sites-available")
        setflags("config", "sites-enabled", "/etc/nginx/sites-enabled")
        setflags("man", "nginx.8", "/usr/share/man/man8/nginx.8")
        setflags("license", "license", "/usr/share/licenses/nginx/LICENSE")
        setflags("systemd", "nginx.service", "/etc/systemd/system/nginx.service")
    end,

}
