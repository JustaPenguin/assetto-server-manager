json = require "json"
utils = require "utils"

-- these are lua hooks related to championships, for help please view lua_readme.md!
-- there are some example functions here to give you an idea of what is possible, feel free to write your own!
-- if you do and think other people would be interested in them consider making a pull request at https://github.com/JustaPenguin/assetto-server-manager

-- called when a championship is started from the UI, before it is started on the server
function onChampionshipEventStart(encodedEvent, encodedChampionship, encodedClassStandings, encodedEntryList)
    -- Decode block, you probably shouldn't touch these!
    local event = json.decode(encodedEvent)
    local championship = json.decode(encodedChampionship)
    local standings = json.decode(encodedClassStandings)
    local entryList = json.decode(encodedEntryList)

    -- Uncomment these lines and run the function (start any championship event, including practice events) to print out
    -- the structure of each object.
    --print("Event:", dump(event))
    --print("Championship:", dump(championship)) --championships can get pretty huge, this might exceed terminal limit
    --print("Standings:", dump(standings))

    -- Function block NOTE: this hook BLOCKS, make sure your functions don't loop forever!
    -- uncomment functions to enable them!

    --entryList = addBallastFromChampionshipPosition(entryList, standings, 50)
    --entryList = addBallastFromChampionshipEventPosition(championship, entryList, 100, 1, false)

    -- Encode block, you probably shouldn't touch these either!
    return json.encode(championship), json.encode(event), json.encode(entryList)
end

-- called when any CHAMPIONSHIP event is scheduled
function onChampionshipEventSchedule(encodedEvent, encodedChampionship, encodedClassStandings)
    -- Decode block, you probably shouldn't touch these!
    local event = json.decode(encodedEvent)
    local championship = json.decode(encodedChampionship)
    local standings = json.decode(encodedClassStandings)

    -- Uncomment these lines and run the function (start any event) to print out the structure of each object.
    --print("Event:", dump(event))
    --print("Championship:", dump(championship)) --championships can get pretty huge, this might exceed terminal limit
    --print("Standings:", dump(standings))

    -- Function block NOTE: this hook BLOCKS, make sure your functions don't loop forever!


    -- Encode block, you probably shouldn't touch these either!
    return json.encode(championship), json.encode(event)
end

-- add ballast to drivers for the championship event based on their current championship position
function addBallastFromChampionshipPosition(entryList, standings, maxBallast)
    -- loop over each championship class
    for className,classStandings in pairs(standings) do
        -- loop over the standings for the class
        for pos,standing in pairs(classStandings) do
            -- loop over cars in the entry list
            for carID, entrant in pairs(entryList) do
                -- if standing and entrant guids match
                if entrant["GUID"] == standing["Car"]["Driver"]["Guid"] then
                    -- add ballast based on championship position
                    entrant["Ballast"] = math.floor(maxBallast/(pos))
                end
            end
        end
    end

    return entryList
end

-- add ballast to drivers for the championship event based on the results of some event
function addBallastFromChampionshipEventPosition(championship, entryList, maxBallast, nthMostRecentEvent, reverseRace)
    table.sort(championship["Events"], function (left, right)
        return left["CompletedTime"] > right["CompletedTime"]
    end )

    -- event to apply ballast from is now nth in championship["Events"]
    for sessionType,session in pairs(championship["Events"][nthMostRecentEvent]["Sessions"]) do
        if (not (session["CompletedTime"] == "0001-01-01T00:00:00Z")) and (sessionType == "RACE" and (not reverseRace)) or (sessionType == "RACEx2" and reverseRace) then
            -- start at maxBallast, decrease for each driver by diff
            local diff = 9
            local ballast = maxBallast

            for pos,result in pairs(session["Results"]["Result"]) do
                -- find our entrant and apply ballast
                for carID, entrant in pairs(entryList) do
                    -- if standing and entrant guids match
                    if entrant["GUID"] == result["DriverGuid"] then
                        -- add ballast based on result in nth most recent event
                        entrant["Ballast"] = ballast

                        print("LUA: applying ", ballast, "kg of ballast to ", result["DriverName"], ". Finished in pos ", pos, " at ", session["Results"]["TrackName"])
                        break
                    end
                end

                ballast = ballast - diff

                -- if you want non-linear ballast, modify the diff here
                if pos == 4 then
                    diff = 6
                end
            end
        end
    end

    return entryList
end

function forceVirtualMirror(event)
    -- keep racing safe, all the time
    event["RaceSetup"]["ForceVirtualMirror"] = 1

    return event
end
