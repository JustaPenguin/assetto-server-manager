utils = require "utils"

-- these are lua hooks related to the manager itself, for help please view lua_readme.md!
-- there are some example functions here to give you an idea of what is possible, feel free to write your own!
-- if you do and think other people would be interested in them consider making a pull request at https://github.com/JustaPenguin/assetto-server-manager

-- called whenever Server Manager is started
function onManagerStart()

    -- Function block NOTE: this hook doesn't block, so it can run constantly alongside the manager if required
    results = notifyLuaInitialised()

end

function notifyLuaInitialised()
    print("Lua plugin hooks have been initialised successfully!")
end