json = require "json"
utils = require "utils"

-- Insert json folder path here, required for getTrackInfo
local jsonPath = "./plugins" -- default should be correct if countryCodes.json is inside plugins folder
local contentPath = "./assetto/content" -- depends on assetto server location

-- these are lua hooks related to the manager itself, for help please view lua_readme.md!
-- there are some example functions here to give you an idea of what is possible, feel free to write your own!
-- if you do and think other people would be interested in them consider making a pull request at https://github.com/JustaPenguin/assetto-server-manager

-- called when any event (including championships/race weekends) is started (for championships this is called AFTER onChampionshipEventStart)
function onEventStart(encodedRaceConfig, encodedServerOpts, encodedEntryList)
    -- Decode block, you probably shouldn't touch these!
    local raceConfig = json.decode(encodedRaceConfig)
    local serverOpts = json.decode(encodedServerOpts)
    local entryList = json.decode(encodedEntryList)

    -- Uncomment these lines and run the function (start any event) to print out the structure of each object.
    --print("Race Config:", utils.dump(raceConfig))
    --print("Server Options:", utils.dump(serverOpts))
    --print("Entry List:", utils.dump(entryList))

    -- Function block NOTE: this hook BLOCKS, make sure your functions don't loop forever!

    -- Uncomment this line to set Weather API On
    -- in order to use the weatherAPI you need to get a free API key from https://openweathermap.org/
    raceConfig, serverOpts = getWeatherForTrack(raceConfig, serverOpts, "get-an-api-key-from-https://openweathermap.org/")

    -- Encode block, you probably shouldn't touch these either!
    return json.encode(entryList), json.encode(serverOpts), json.encode(raceConfig)
end

-- called when any NON CHAMPIONSHIP/RACE WEEKEND event is scheduled
function onEventSchedule(encodedRace)
    -- Decode block, you probably shouldn't touch these!
    local race = json.decode(encodedRace)

    -- Uncomment these lines and run the function (start any event) to print out the structure of each object.
    --print("Race:", utils.dump(race))

    -- Function block NOTE: this hook BLOCKS, make sure your functions don't loop forever!


    -- Encode block, you probably shouldn't touch these either!
    return json.encode(race)
end

-- called when any RACE WEEKEND event is scheduled
function onRaceWeekendEventSchedule(encodedRaceWeekendSession, encodedRaceWeekend)
    -- Decode block, you probably shouldn't touch these!
    local session = json.decode(encodedRaceWeekendSession)
    local raceWeekend = json.decode(encodedRaceWeekend)

    -- Uncomment these lines and run the function (start any event) to print out the structure of each object.
    --print("Race Weekend Session:", utils.dump(session))
    --print("Race Weekend:", utils.dump(raceWeekend))

    -- Function block NOTE: this hook BLOCKS, make sure your functions don't loop forever!

    -- Encode block, you probably shouldn't touch these either!
    return json.encode(raceWeekend), json.encode(session)
end

function getWeatherForTrack(raceConfig, serverOpts, apiKey)
    if apiKey == "get-an-api-key-from-https://openweathermap.org/" then
        print("events.lua: No API key is set, could not activate weather API")

        return raceConfig, serverOpts
    end

    -- Getting location from track_ui.json, make sure to set jsonPath and contentPath
    location = getTrackInfo(raceConfig)
    -- if you want to manually set the location for tracks without location info uncomment this line and set the location, you can download a city list here: http://bulk.openweathermap.org/sample/
    -- location = "Manchester,uk"

    if location == nil then
        --Set location manually if no trackinfo is found
        location = "Manchester,UK"
    end

    return weatherAPI(raceConfig, serverOpts, apiKey)
end

-- get track location from his ui.json file, set dynamic weather on if a location is found
-- if not, use weather in custom race config
function getTrackInfo(raceConfig)
    local track = raceConfig["Track"]
    local layout = raceConfig["TrackLayout"]

    local trackPath = contentPath .. "/tracks/" .. track .. "/ui/" .. layout
    local encodedTrackJson = utils.jsonOpen(trackPath, "ui_track.json")
    local trackJson

    success = pcall(function()
        trackJson = json.decode(encodedTrackJson)
    end)

    if not success then
        print("events.lua: Couldn't decode track UI file: ", trackPath .. "/ui_track.json. Falling back to manual setting")
        return nil
    end

    countryFull = trackJson["country"]
    city = trackJson["city"]

    local encodedCountryCodes = utils.jsonOpen(jsonPath, "countryCodes.json")
    local countryCodes = json.decode(encodedCountryCodes)

    if city == nil or countryFull == nil then
        print("events.lua: No country/city found in track UI file, falling back to manual setting")
        return nil
    end

    if countryCodes[countryFull] == nil then
        print("events.lua: Location was found in track UI file (" .. city .. ", " .. countryFull .. "). But was not found in countryCodes.json, please update! Falling back to manual setting")
        return nil
    end

    location = city .. "," .. countryCodes[countryFull]

    return location
end

-- weather API
function weatherAPI(raceConfig, serverOpts, apiKey)
    -- set the weather based on the current weather at 'location'
    local body, status = httpRequest("http://api.openweathermap.org/data/2.5/weather?q=" .. location .. "&APPID=" .. apiKey, "GET", "")

    -- If location not found in openWeatherMap, stop the function and use weather configured in web manager
    if status >= 400 then
        print("events.lua: Weather API HTTP request returned status ", status, ". Weather API deactivated")
        return raceConfig, serverOpts
    end

    local weatherData = json.decode(body)

    -- wind speed, from m/s to km/h
    raceConfig["WindBaseSpeedMin"] = math.floor(weatherData["wind"]["speed"] * 3.6)
    raceConfig["WindBaseSpeedMax"] = raceConfig["WindBaseSpeedMin"] + 2
    -- wind angle
    raceConfig["WindBaseDirection"] = weatherData["wind"]["deg"]
    raceConfig["WindVariationDirection"] = 5

    -- there should only be one weather, but we'll apply to all just in case
    for name, weather in pairs(raceConfig["Weather"]) do

        -- ambient temp, from Kelvin to Degrees Celcius
        weather["BaseTemperatureAmbient"] = math.floor(weatherData["main"]["temp"] - 273)
        weather["VariationAmbient"] = 0

        -- road temp
        if weatherData["dt"] > weatherData["sys"]["sunrise"] and weatherData["dt"] < weatherData["sys"]["sunset"] then
            -- the sun is up, base road temp should be a bit higher than ambient
            weather["BaseTemperatureRoad"] = 4
        else
            -- sun is down, base road temp is lower than ambient (large assumption, definitely wrong, please improve!)
            weather["BaseTemperatureRoad"] = 0
        end

        weather["VariationRoad"] = 1

        -- weather codes can be found here: https://openweathermap.org/weather-conditions
        local w = weatherData["weather"][1]["id"]

        if raceConfig["IsSol"] == 1 then
            -- with sol (recommended)

            -- set time of day
            weather["CMWFUseCustomTime"] = 1
            weather["CMWFXTime"] = 0
            weather["CMWFXUseCustomDate"] = 1
            weather["CMWFXDateUnModified"] = weatherData["dt"]
            weather["CMWFXDate"] = weatherData["dt"] - (3600 * 5 * weather["CMWFXTimeMulti"]) -- don't ask

            -- force time 5 hour before sunset (18000) to prevent night
            --weather["CMWFXDate"] = (weatherData["sys"]["sunset"] - 18000) + (weatherData["timezone"]) - (3600 * 5 * weather["CMWFXTimeMulti"]) -- don't ask

            -- set graphics (comment this and uncomment the block bellow for no rain)
            if w == 800 then
                weather["CMGraphics"] = "sol_01_CLear";
                weather["CMWFXType"] = 15;
            elseif w == 801 then
                weather["CMGraphics"] = "sol_02_Few Clouds";
                weather["CMWFXType"] = 16
            elseif w == 802 then
                weather["CMGraphics"] = "sol_03_Scattered Clouds";
                weather["CMWFXType"] = 17
                --Do not uncomment elseif w ==  then weather["CMGraphics"] = "sol_04_Windy"; weather["CMWFXType"] = 31 --no real weather for windy
            elseif w == 803 then
                weather["CMGraphics"] = "sol_05_Broken Clouds";
                weather["CMWFXType"] = 18
            elseif w == 804 then
                weather["CMGraphics"] = "sol_06_Overcast";
                weather["CMWFXType"] = 19
            elseif w == 701 then
                weather["CMGraphics"] = "sol_11_Mist";
                weather["CMWFXType"] = 21
            elseif w == 741 then
                weather["CMGraphics"] = "sol_12_Fog";
                weather["CMWFXType"] = 20
            elseif w == 721 then
                weather["CMGraphics"] = "sol_21_Haze";
                weather["CMWFXType"] = 23
            elseif w == 731 then
                weather["CMGraphics"] = "sol_22_Dust";
                weather["CMWFXType"] = 25
            elseif w == 751 then
                weather["CMGraphics"] = "sol_23_Sand";
                weather["CMWFXType"] = 24
            elseif w == 711 then
                weather["CMGraphics"] = "sol_24_Smoke";
                weather["CMWFXType"] = 22
            elseif w == 300 then
                weather["CMGraphics"] = "sol_31_Light Drizzle";
                weather["CMWFXType"] = 3
            elseif w == 301 then
                weather["CMGraphics"] = "sol_32_Drizzle";
                weather["CMWFXType"] = 4
            elseif w >= 302 and w <= 321 then
                weather["CMGraphics"] = "sol_33_Heavy Drizzle";
                weather["CMWFXType"] = 5
            elseif w == 500 then
                weather["CMGraphics"] = "sol_34_Light Rain";
                weather["CMWFXType"] = 6
            elseif w == 501 then
                weather["CMGraphics"] = "sol_35_Rain";
                weather["CMWFXType"] = 7
            elseif w >= 502 and w <= 531 then
                weather["CMGraphics"] = "sol_36_Heavy Rain";
                weather["CMWFXType"] = 8
            elseif w == 200 or w == 210 or w == 230 then
                weather["CMGraphics"] = "sol_41_Light Thunderstorm";
                weather["CMWFXType"] = 0
            elseif w == 201 or w == 211 or w == 231 then
                weather["CMGraphics"] = "sol_42_Thunderstorm";
                weather["CMWFXType"] = 1
            elseif w == 202 or w == 212 or w == 221 or w == 232 then
                weather["CMGraphics"] = "sol_43_Heavy Thunderstorm";
                weather["CMWFXType"] = 2
            elseif w == 771 then
                weather["CMGraphics"] = "sol_44_Squalls";
                weather["CMWFXType"] = 26
            elseif w == 781 then
                weather["CMGraphics"] = "sol_45_Tornado";
                weather["CMWFXType"] = 27
                --Do not uncomment elseif w ==  then weather["CMGraphics"] = "sol_46_Hurricane"; weather["CMWFXType"] = 28 --no real weather for hurricane
            elseif w == 600 or w == 620 then
                weather["CMGraphics"] = "sol_51_Light Snow";
                weather["CMWFXType"] = 9
            elseif w == 601 or w == 621 then
                weather["CMGraphics"] = "sol_52_Snow";
                weather["CMWFXType"] = 10
            elseif w == 602 or w == 622 then
                weather["CMGraphics"] = "sol_53_Heavy Snow";
                weather["CMWFXType"] = 11
            elseif w == 611 or w == 615 then
                weather["CMGraphics"] = "sol_54_Light Sleet";
                weather["CMWFXType"] = 12
            elseif w == 612 or w == 616 then
                weather["CMGraphics"] = "sol_55_Sleet";
                weather["CMWFXType"] = 13
            elseif w == 613 then
                weather["CMGraphics"] = "sol_56_Heavy Sleet";
                weather["CMWFXType"] = 14
                --Do not uncomment elseif w ==  then weather["CMGraphics"] = "sol_57_Hail"; weather["CMWFXType"] = 32 --no real weather for hail
            end

            -- set graphics no rain (comment the block above and uncomment this one for no rain)
            --if     w == 800 then weather["CMGraphics"] = "sol_01_CLear"; weather["CMWFXType"] = 15;
            --elseif w == 801 then weather["CMGraphics"] = "sol_02_Few Clouds"; weather["CMWFXType"] = 16
            --elseif w == 802 then weather["CMGraphics"] = "sol_03_Scattered Clouds"; weather["CMWFXType"] = 17
            --Do not uncomment elseif w ==  then weather["CMGraphics"] = "sol_04_Windy"; weather["CMWFXType"] = 31 --no real weather for windy
            --elseif w == 803 then weather["CMGraphics"] = "sol_05_Broken Clouds"; weather["CMWFXType"] = 18
            --elseif w == 804 then weather["CMGraphics"] = "sol_06_Overcast"; weather["CMWFXType"] = 19
            --elseif w == 701 then weather["CMGraphics"] = "sol_11_Mist"; weather["CMWFXType"] = 21
            --elseif w == 741 then weather["CMGraphics"] = "sol_12_Fog"; weather["CMWFXType"] = 20
            --elseif w == 721 then weather["CMGraphics"] = "sol_21_Haze"; weather["CMWFXType"] = 23
            --elseif w == 731 then weather["CMGraphics"] = "sol_22_Dust"; weather["CMWFXType"] = 25
            --elseif w == 751 then weather["CMGraphics"] = "sol_23_Sand"; weather["CMWFXType"] = 24
            --elseif w == 711 then weather["CMGraphics"] = "sol_24_Smoke"; weather["CMWFXType"] = 22
            --elseif w == 300 then weather["CMGraphics"] = "sol_05_Broken Clouds"; weather["CMWFXType"] = 18
            --elseif w == 301 then weather["CMGraphics"] = "sol_06_Overcast"; weather["CMWFXType"] = 19
            --elseif w >= 302 and w <= 321 then weather["CMGraphics"] = "sol_24_Smoke"; weather["CMWFXType"] = 22
            --elseif w == 500 then weather["CMGraphics"] = "sol_05_Broken Clouds"; weather["CMWFXType"] = 18
            --elseif w == 501 then weather["CMGraphics"] = "sol_06_Overcast"; weather["CMWFXType"] = 19
            --elseif w >= 502 and w <= 531 then weather["CMGraphics"] = "sol_24_Smoke"; weather["CMWFXType"] = 22
            --elseif w == 200 or w == 210 or w == 230 then weather["CMGraphics"] = "sol_06_Overcast"; weather["CMWFXType"] = 19
            --elseif w == 201 or w == 211 or w == 231 then weather["CMGraphics"] = "sol_24_Smoke"; weather["CMWFXType"] = 22
            --elseif w == 202 or w == 212 or w == 221 or w == 232 then weather["CMGraphics"] = "sol_24_Smoke"; weather["CMWFXType"] = 22
            --elseif w == 771 then weather["CMGraphics"] = "sol_44_Squalls"; weather["CMWFXType"] = 26
            --elseif w == 781 then weather["CMGraphics"] = "sol_24_Smoke"; weather["CMWFXType"] = 22
            --Do not uncomment elseif w ==  then weather["CMGraphics"] = "sol_46_Hurricane"; weather["CMWFXType"] = 28 --no real weather for hurricane
            --elseif w == 600 or w == 620 then weather["CMGraphics"] = "sol_05_Broken Clouds"; weather["CMWFXType"] = 18
            --elseif w == 601 or w == 621 then weather["CMGraphics"] = "sol_12_Fog"; weather["CMWFXType"] = 20
            --elseif w == 602 or w == 622 then weather["CMGraphics"] = "sol_24_Smoke"; weather["CMWFXType"] = 22
            --elseif w == 611 or w == 615 then weather["CMGraphics"] = "sol_05_Broken Clouds"; weather["CMWFXType"] = 18
            --elseif w == 612 or w == 616 then weather["CMGraphics"] = "sol_12_Fog"; weather["CMWFXType"] = 20
            --elseif w == 613 then weather["CMGraphics"] = "sol_06_Overcast"; weather["CMWFXType"] = 19
            --Do not uncomment elseif w ==  then weather["CMGraphics"] = "sol_57_Hail"; weather["CMWFXType"] = 32 --no real weather for hail
            --end

            weather["Graphics"] = weather["CMGraphics"] .. "_type=" .. weather["CMWFXType"] .. "_time=0_mult=" .. weather["CMWFXTimeMulti"] .. "_start=" .. weather["CMWFXDate"]
        else
            -- without Sol (not recommended)
            -- you could set sun angle from time of day here, I'm not going to though (just use Sol)

            -- set graphics
            if w == 800 then
                weather["Graphics"] = "3_clear"
            elseif w == 801 then
                weather["Graphics"] = "4_mid_clear"
            elseif w == 802 then
                weather["Graphics"] = "5_light_clouds"
            elseif w == 803 then
                weather["Graphics"] = "6_mid_clouds"
            elseif w == 804 then
                weather["Graphics"] = "7_heavy_clouds"
            elseif w == 741 then
                weather["Graphics"] = "2_light_fog"
            end
        end

    end

    -- Add text to server name to indicate this has been done
    serverOpts["Name"] = serverOpts["Name"] .. " | Weather Live From " .. weatherData["name"]

    return raceConfig, serverOpts
end

--{ Response json from openweathermap looks like this
--	"coord": {
--		"lon": -0.13,
--		"lat": 51.51
--	},
--	"weather": [{
--		"id": 803,
--		"main": "Clouds",
--		"description": "broken clouds",
--		"icon": "04d"
--	}],
--	"base": "stations",
--	"main": {
--		"temp": 281.13,
--		"pressure": 998,
--		"humidity": 66,
--		"temp_min": 280.15,
--		"temp_max": 282.15
--	},
--	"visibility": 10000,
--	"wind": {
--		"speed": 3.1,
--		"deg": 240
--	},
--	"rain": {},
--	"clouds": {
--		"all": 67
--	},
--	"dt": 1573656582,
--	"sys": {
--		"type": 1,
--		"id": 1414,
--		"country": "GB",
--		"sunrise": 1573629270,
--		"sunset":  1573661710
--	},
--	"timezone": 0,
--	"id": 2643743,
--	"name": "London",
--	"cod": 200
--}
