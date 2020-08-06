json = require "json"
utils = require "utils"

-- there are lua hooks related to live events that take place whilst an event is running on the server
-- for help please view lua_readme.md!
-- there are some example functions here to give you an idea of what is possible, feel free to write your own!
-- if you do and think other people would be interested in them consider making a pull request at https://github.com/JustaPenguin/assetto-server-manager

-- called whenever a chat is sent in-game, from the live timings page or from a lua script
function onChat(encodedChat)
    -- Decode block, you probably shouldn't touch these!
    local chat = json.decode(encodedChat)

    -- Uncomment these lines and run the function (send a chat message) to print out the structure of each object.
    -- Note that the chat object contains more information than just the message text, it has name etc. too
    --print("Chat Message:", utils.dump(chat))

    -- Function block NOTE: this hook doesn't block, so it can run constantly alongside the manager if required
end
