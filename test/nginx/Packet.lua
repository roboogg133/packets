return {
    package = {
        name = "nginx",
        version = "1.29.3",
        maintainer = "robogg133",
        description =
        [[nginx ("engine x") is an HTTP web server, reverse proxy, content cache, load balancer, TCP/UDP proxy server, and mail proxy server. Originally written by Igor Sysoev and distributed under the 2-clause BSD License. Enterprise distributions, commercial support and training are available from F5, Inc.]],
        serial = 0,
        pageurl = "https://nginx.org",
        LICENSE = {
            "BSD-2-Clause"
        },

        plataforms = {
            windows = {
                arch = { "amd64", "386" },
                sources = {
                    {
                        url = "https://nginx.org/download/nginx-1.29.3.zip",
                        method = "GET",
                        sha256 = { "afa2fde9fdf0ac64b91a17dcd34100ac557a3ff8e6154eeb0eeae7aa8e5bbc2d" }
                    }
                }
            },
            linux = {
                arch = { "amd64", "386", "arm64" },
                sources = {
                    {
                        url = "https://nginx.org/download/nginx-1.29.3.tar.gz",
                        method = "GET",
                        sha256 = { "9befcced12ee09c2f4e1385d7e8e21c91f1a5a63b196f78f897c2d044b8c9312" }

                    }
                },
            }
        },

        sources = {},
        dependencies = {
            build = {
                "gcc",
                "cmake",
                "make"
            },
            runtime = {},
            conflicts = {}
        }

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

        install("nginx.service", "/etc/systemd/system/nginx.service")
        os.chdir(pathjoin(SOURCESDIR, uncompressedname))

        os.chmod("objs/nginx", 755)
        install("objs/nginx", pathjoin(BIN_DIR, "nginx"))

        os.mkdir("nginx", 755)
        install("nginx", "/usr/local/nginx")
        install("nginx", "/etc/nginx")

        install("conf/koi-win", "/etc/nginx/koi-win")
        install("conf/koi-utf", "/etc/nginx/koi-utf")
        install("conf/win-utf", "/etc/nginx/win-utf")

        install("conf/mime.types", "/etc/nginx/mime.types")
        install("conf/mime.types", "/etc/nginx/mime.types.default")

        install("conf/fastcgi_params", "/etc/nginx/fastcgi_params")
        install("conf/fastcgi_params", "/etc/nginx/fastcgi_params.default")

        install("conf/fastcgi.conf", "/etc/nginx/fastcgi.conf")
        install("conf/fastcgi.conf", "/etc/nginx/fastcgi.conf.default")

        install("conf/uwsgi_params", "/etc/nginx/uwsgi_params")
        install("conf/uwsgi_params", "/etc/nginx/uwsgi_params.default")


        install("conf/scgi_params", "/etc/nginx/scgi_params")
        install("conf/scgi_params", "/etc/nginx/scgi_params.default")

        install("conf/nginx.conf", "/etc/nginx/nginx.conf")
        install("conf/nginx.conf", "/etc/nginx/nginx.conf.default")

        install("html", "/usr/share/nginx/html")

        install("LICENSE", "/usr/share/licenses/nginx/LICENSE")

        install("man/nginx.8", "/usr/share/man/man8/nginx.8")

        os.mkdir("logs", 755)
        install("logs", "/etc/nginx/logs")

        setflags("bin", "nginx", pathjoin(BIN_DIR, "nginx"))
        setflags("config", "main", "/etc/nginx/nginx.conf")
        setflags("config", "sites-available", "/etc/nginx/sites-available")
        setflags("config", "sites-enabled", "/etc/nginx/sites-enabled")
        setflags("man", "nginx.8", "/usr/share/man/man8/nginx.8")
        setflags("license", "license", "/usr/share/licenses/nginx/LICENSE")
        setflags("systemd", "nginx.service", "/etc/systemd/system/nginx.service")
    end,

}
