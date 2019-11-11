json = require "json"

-- these are lua hooks related to championships, for help please view luaHelp.md!
-- there are some example functions here to give you an idea of what is possible, feel free to write your own!
-- if you do and think other people would be interested in them consider making a pull request at https://github.com/cj123/assetto-server-manager

-- called when a championship is started from the UI, before it is started on the server
function championshipEventStart(encodedEvent, encodedChampionship, encodedClassStandings)
    -- Decode block, you probably shouldn't touch these!
    local event = json.decode(encodedEvent)
    local championship = json.decode(encodedChampionship)
    local standings = json.decode(encodedClassStandings)

    -- Uncomment these lines and run the function (start any championship event, including practice events) to print out
    -- the structure of each object.
    --print("Event:", dump(event))
    --print("Championship:", dump(championship)) --championships can get pretty huge, this might exceed terminal limit
    --print("Standings:", dump(standings))

    -- Function block
    event = addBallastFromChampionshipPosition(event, standings)

    -- Encode block, you probably shouldn't touch these either!
    return json.encode(event)
end

function addBallastFromChampionshipPosition(event, standings)
    -- loop over each championship class
    for className,classStandings in pairs(standings) do
        -- loop over the standings for the class
        for pos,standing in pairs(classStandings) do
            -- loop over cars in the entry list
            for carID, entrant in pairs(event["EntryList"]) do
                -- if standing and entrant guids match
                if entrant["GUID"] == standing["Car"]["Driver"]["Guid"] then
                    -- add ballast based on championship position
                    entrant["Ballast"] = math.floor(20/(pos))
                end
            end
        end
    end

    return event
end

function forceVirtualMirror(event)
    -- keep racing safe, all the time
    event["RaceSetup"]["ForceVirtualMirror"] = 1

    return event
end

function dump(o)
    if type(o) == 'table' then
        local s = '{ '
        for k,v in pairs(o) do
            if type(k) ~= 'number' then k = '"'..k..'"' end
            s = s .. '['..k..'] = ' .. dump(v) .. ',' .. "\n"
        end
        return s .. '} '
    else
        return tostring(o)
    end
end