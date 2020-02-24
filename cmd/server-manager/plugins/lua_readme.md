##A Vaguely Acceptable Lua Plugin Guide
######*Written by Hecrer*

Welcome to my short guide on using Lua script plugins with Server Manager. If you're
at all accustomed to Lua then you probably already know it better than I do, if not
you may find this document useful!

*Lua plugins are a premium Server Manager feature! If you want to use them please consider supporting us and 
the development of Server Manager by [going premium](https://paypal.me/JustaPenguinUK) ($10 or more) 
or [renting a server](https://assettocorsaservers.com).*

First of all it's important to have a basic understanding of Lua syntax and standards, there's an official 
get started guide here that I highly recommend checking out first:
 
[Lua Get Started Guide](https://www.lua.org/pil/1.html)

In order to activate Lua plugins for SM you need to set the following in your ```config.yml```:

```yaml
################################################################################
#
#  lua config - configure lua plugins
#
################################################################################
lua:
  # lua plugins allow you to run custom lua scripts through hooks with server
  # manager! If you're interested have a look at the server-manager/plugins
  # folder to see some examples!
  enabled: true
```

Once you've got the basics down we can have a look at how Server Manager (SM) interacts with our Lua plugins. 
It's worth noting that SM automatically adds the ```server-manager/plugins``` folder to your ```LUA_PATH``` when it initialises.
If you want to import Lua files from other locations then you can add them to your ```LUA_PATH``` manually.

Here is an example of the program flow for the ```onEventStart``` Lua hook:

```
Event Start button is pressed in the SM UI
Backend calls Lua function onEventStart and sends the configured event encoded as json
Lua code can be used to modify the event
Lua encodes the event back to json and returns it to the backend
Backend starts the event
```

All of the plugins have a flow similar to this, and all of them receive json encoded data from the backend and pass it 
back when they are done with it. All of the json encoding/decoding is already set up in the lua functions, don't touch 
these or things will break!

The json data is decoded into a Lua ```Table```, a data type that you can learn about in the get started guide linked above. 
You can modify anything you find in these tables, but please remember to keep the ```type``` the same. For example don't 
start encoding ```float``` values where you initially received ```int``` values. Lua isn't strictly typed, but Golang 
(the programming language that our backend runs in) *is*, so for it to function properly types must not change! 

Also remember that you aren't just limited to hooks, Lua has access to the filesystem and is capable of reading and 
modifying files completely by itself (you can use ```onManagerStart``` in ```manager.lua``` to start Lua scripts that run 
independently), so there's a huge range of possibilities!

I'm excited to see what people start using Lua plugins for, and hope that this guide is at least a little bit useful! 
If you make something cool please share it with the 
community by making a pull request on [Github](https://github.com/JustaPenguin/assetto-server-manager).

**Now the best way to learn is to get stuck in! Have a look at the example functions in the existing Lua files, they 
are well documented and should get you on the right track. If you have any further questions please join our 
[Discord](https://discordapp.com/invite/6DGKJzB)!**