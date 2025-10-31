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

    end,
    
    install  = function() -- required 
        print("goku")
    end,

}