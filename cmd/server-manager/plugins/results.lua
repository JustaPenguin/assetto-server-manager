json = require "json"
utils = require "utils"

-- these are lua hooks related to results, for help please view lua_readme.md!
-- there are some example functions here to give you an idea of what is possible, feel free to write your own!
-- if you do and think other people would be interested in them consider making a pull request at https://github.com/JustaPenguin/assetto-server-manager

-- called whenever results are loaded, including by championships and race weekends
function onResultsLoad(encodedResults)
    -- Decode block, you probably shouldn't touch these!
    local results = json.decode(encodedResults)

    -- Uncomment these lines and run the function (load a results file) to print out the structure of each object.
    --print("Results:", utils.dump(results))

    -- Function block NOTE: this hook BLOCKS, make sure your functions don't loop forever!
    -- uncomment functions to enable them!
    --results = autoDisqualifyForCollisions(results, 140.0, 20)

    -- Encode block, you probably shouldn't touch these either!
    return json.encode(results)
end

-- disqualify a driver if they have a collision faster than minSpeed or more total collisions than maxCollisions
function autoDisqualifyForCollisions(results, minSpeed, maxCollisions)
    -- there may not be any events, Result may be empty on list pages
    if results["Result"] == nil or results["Events"] == nil then
        return results
    end

    for k,resultTable in pairs(results["Result"]) do

        local numCollisions = 0

        for i,eventTable in pairs(results["Events"]) do
            -- this collision involved the current driver
            if eventTable["Driver"]["Guid"] == resultTable["DriverGuid"] then
                -- count the number of collisions
                numCollisions = numCollisions + 1

                -- if this collision is faster than the minSpeed, or max collisions have been exceeded
                if eventTable["ImpactSpeed"] > minSpeed or numCollisions > maxCollisions then
                    -- disqualify the driver
                    resultTable["Disqualified"] = true
                end
            end
        end
    end

    -- sort disqualified drivers to the back of the results list
    table.sort(results["Result"], sortDisqualifiedToBack)

    return results
end

function sortDisqualifiedToBack (a, b)
    if a["Disqualified"] and not b["Disqualified"] then
        return false
    elseif not a["Disqualified"] and b["Disqualified"] then
        return true
    else
        return false
    end
end